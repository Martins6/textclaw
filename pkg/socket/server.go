package socket

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/textclaw/textclaw/internal/daemon/listener"
	"github.com/textclaw/textclaw/internal/database"
	"github.com/textclaw/textclaw/internal/embedding"
)

type Server struct {
	socketPath       string
	adapter          listener.Adapter
	listener         net.Listener
	db               *database.DB
	embeddingService *embedding.EmbeddingService
}

func NewServer(socketPath string, adapter listener.Adapter, db *database.DB) *Server {
	if socketPath == "" {
		socketPath = "/var/run/textclaw/textclaw.sock"
	}
	return &Server{
		socketPath: socketPath,
		adapter:    adapter,
		db:         db,
	}
}

func (s *Server) SetEmbeddingService(es *embedding.EmbeddingService) {
	s.embeddingService = es
}

func (s *Server) Start() error {
	dir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to remove existing socket: %v", err)
	}

	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	s.listener = l
	if err := os.Chmod(s.socketPath, 0777); err != nil {
		log.Printf("Warning: failed to chmod socket: %v", err)
	}

	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			log.Printf("Socket accept error: %v", err)
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	var rawMsg json.RawMessage
	if err := json.NewDecoder(conn).Decode(&rawMsg); err != nil {
		log.Printf("Failed to decode message: %v", err)
		return
	}

	var baseMsg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawMsg, &baseMsg); err != nil {
		log.Printf("Failed to parse message type: %v", err)
		return
	}

	if baseMsg.Type == "context_search" {
		s.handleContextSearch(conn, rawMsg)
	} else {
		s.handleNotify(conn, rawMsg)
	}
}

func (s *Server) handleNotify(conn net.Conn, rawMsg json.RawMessage) {
	var msg NotifyMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		log.Printf("Failed to decode notify message: %v", err)
		return
	}

	log.Printf("Received notify from workspace %s: %s", msg.WorkspaceID, msg.Content)

	if s.adapter == nil {
		log.Printf("No adapter configured, cannot send message")
		return
	}

	target := msg.Target
	if target == "" {
		target = msg.ChatID
	}

	if err := s.adapter.Send(target, msg.Content); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func (s *Server) handleContextSearch(conn net.Conn, rawMsg json.RawMessage) {
	var msg ContextSearchMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		log.Printf("Failed to decode context search message: %v", err)
		return
	}

	log.Printf("Received context search from workspace %s: type=%s query=%s",
		msg.WorkspaceID, msg.SearchType, msg.Query)

	var results []database.SearchResult
	var err error

	switch strings.ToLower(msg.SearchType) {
	case "semantic", "similar":
		if s.embeddingService == nil {
			err = fmt.Errorf("embedding service not initialized")
		} else {
			embedding, embErr := s.embeddingService.GenerateEmbedding(msg.Query)
			if embErr != nil {
				err = fmt.Errorf("failed to generate embedding: %w", embErr)
			} else {
				results, err = database.SearchBySimilarity(s.db, msg.WorkspaceID, embedding, msg.Limit)
			}
		}
	case "keyword", "search", "find":
		results, err = database.SearchByKeyword(s.db, msg.WorkspaceID, msg.Query, msg.Limit)
	case "recent":
		messages, msgErr := database.GetMessages(s.db, msg.WorkspaceID, msg.Limit)
		if msgErr != nil {
			err = msgErr
		} else {
			for _, m := range messages {
				results = append(results, database.SearchResult{
					MessageID:   m.ID,
					WorkspaceID: m.WorkspaceID,
					Content:     m.Content,
					Timestamp:   m.Timestamp,
					Similarity:  0,
				})
			}
		}
	default:
		err = fmt.Errorf("unknown search type: %s", msg.SearchType)
	}

	if err != nil {
		errMsg, _ := json.Marshal(map[string]string{"error": err.Error()})
		conn.Write(errMsg)
		return
	}

	response, _ := json.Marshal(results)
	conn.Write(response)
}

func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

type NotifyMessage struct {
	WorkspaceID string `json:"workspace_id"`
	Content     string `json:"content"`
	Target      string `json:"target,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
	Urgent      bool   `json:"urgent,omitempty"`
}

type ContextSearchMessage struct {
	Type        string `json:"type"`
	WorkspaceID string `json:"workspace_id"`
	SearchType  string `json:"search_type"`
	Query       string `json:"query"`
	Limit       int    `json:"limit"`
}
