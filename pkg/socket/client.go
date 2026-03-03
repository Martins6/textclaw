package socket

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/Martins6/textclaw/internal/database"
)

type Client struct {
	socketPath string
}

func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = "/var/run/textclaw/textclaw.sock"
	}
	return &Client{socketPath: socketPath}
}

func (c *Client) SendNotify(workspaceID, content, target string, urgent bool) error {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	msg := NotifyMessage{
		WorkspaceID: workspaceID,
		Content:     content,
		Target:      target,
		Urgent:      urgent,
	}

	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	return nil
}

func (c *Client) SendNotifyFromDB(db *database.DB, workspaceID, content string, urgent bool) error {
	contacts, err := database.GetWorkspaceContacts(db, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get contacts: %w", err)
	}

	if len(contacts) == 0 {
		return fmt.Errorf("no contacts found for workspace %s", workspaceID)
	}

	target := contacts[0].ID
	return c.SendNotify(workspaceID, content, target, urgent)
}

func (c *Client) SendHeartbeatNotify(workspaceID, content, target string, suppressNoUpdates bool) error {
	if suppressNoUpdates && content == "No updates" {
		return nil
	}
	return c.SendNotify(workspaceID, content, target, false)
}

func (c *Client) ContextSearch(workspaceID, searchType, query string, limit int) ([]database.SearchResult, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	if limit <= 0 {
		limit = 10
	}

	msg := ContextSearchMessage{
		Type:        "context_search",
		WorkspaceID: workspaceID,
		SearchType:  searchType,
		Query:       query,
		Limit:       limit,
	}

	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	var response map[string]string
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if errMsg, ok := response["error"]; ok {
		return nil, fmt.Errorf("server error: %s", errMsg)
	}

	conn.Close()
	conn, err = net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect: %w", err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	var results []database.SearchResult
	if err := json.NewDecoder(conn).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	return results, nil
}
