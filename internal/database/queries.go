package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Contact struct {
	ID          string
	WorkspaceID string
	Role        string
	CreatedAt   time.Time
}

type Workspace struct {
	ID          string
	ContainerID *string
	SessionID   *string
	CreatedAt   time.Time
}

type Message struct {
	ID          int64
	WorkspaceID string
	ContactID   string
	Content     string
	ContentType string
	Direction   string
	Timestamp   time.Time
}

func InitDB(path string) (*DB, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", absPath+"?_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(1)

	return &DB{DB: db, path: absPath}, nil
}

func InitSchema(db *DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS contacts (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		role TEXT DEFAULT 'user' CHECK(role IN ('main', 'admin', 'user')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS workspaces (
		id TEXT PRIMARY KEY,
		container_id TEXT,
		session_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workspace_id TEXT NOT NULL,
		contact_id TEXT NOT NULL,
		content TEXT NOT NULL,
		content_type TEXT DEFAULT 'text' CHECK(content_type IN ('text', 'non-text')),
		direction TEXT NOT NULL CHECK(direction IN ('incoming', 'outgoing')),
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_workspace ON messages(workspace_id);
	CREATE INDEX IF NOT EXISTS idx_messages_contact ON messages(contact_id);
	CREATE INDEX IF NOT EXISTS idx_contacts_workspace ON contacts(workspace_id);
	`
	_, err := db.Exec(schema)
	return err
}

func CreateContact(db *DB, id, workspaceID, role string) error {
	_, err := db.Exec(
		"INSERT OR REPLACE INTO contacts (id, workspace_id, role) VALUES (?, ?, ?)",
		id, workspaceID, role,
	)
	return err
}

func GetContact(db *DB, id string) (*Contact, error) {
	var c Contact
	err := db.QueryRow(
		"SELECT id, workspace_id, role, created_at FROM contacts WHERE id = ?",
		id,
	).Scan(&c.ID, &c.WorkspaceID, &c.Role, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func CreateWorkspace(db *DB, id string) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO workspaces (id) VALUES (?)",
		id,
	)
	return err
}

func GetWorkspace(db *DB, id string) (*Workspace, error) {
	var w Workspace
	err := db.QueryRow(
		"SELECT id, container_id, session_id, created_at FROM workspaces WHERE id = ?",
		id,
	).Scan(&w.ID, &w.ContainerID, &w.SessionID, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func GetWorkspaceSession(db *DB, workspaceID string) (string, error) {
	var sessionID sql.NullString
	err := db.QueryRow(
		"SELECT session_id FROM workspaces WHERE id = ?",
		workspaceID,
	).Scan(&sessionID)
	if err != nil {
		return "", err
	}
	return sessionID.String, nil
}

func UpdateWorkspaceSession(db *DB, workspaceID, sessionID string) error {
	_, err := db.Exec(
		"UPDATE workspaces SET session_id = ? WHERE id = ?",
		sessionID, workspaceID,
	)
	return err
}

func SaveMessage(db *DB, msg *Message) error {
	_, err := db.Exec(
		`INSERT INTO messages (workspace_id, contact_id, content, content_type, direction)
		 VALUES (?, ?, ?, ?, ?)`,
		msg.WorkspaceID, msg.ContactID, msg.Content, msg.ContentType, msg.Direction,
	)
	return err
}

func GetMessages(db *DB, workspaceID string, limit int) ([]Message, error) {
	rows, err := db.Query(
		`SELECT id, workspace_id, contact_id, content, content_type, direction, timestamp
		 FROM messages WHERE workspace_id = ? ORDER BY timestamp DESC LIMIT ?`,
		workspaceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.WorkspaceID, &m.ContactID, &m.Content, &m.ContentType, &m.Direction, &m.Timestamp)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func GetWorkspaceContacts(db *DB, workspaceID string) ([]Contact, error) {
	rows, err := db.Query(
		"SELECT id, workspace_id, role, created_at FROM contacts WHERE workspace_id = ?",
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Role, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func GetAllWorkspaces(db *DB) ([]Workspace, error) {
	rows, err := db.Query("SELECT id, container_id, session_id, created_at FROM workspaces")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.ContainerID, &w.SessionID, &w.CreatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

type SearchResult struct {
	MessageID   int64
	WorkspaceID string
	Content     string
	Timestamp   time.Time
	Similarity  float64
}

func SaveMessageEmbedding(db *DB, messageID int64, workspaceID string, embedding []float32) error {
	embeddingJSON := vectorToJSON(embedding)
	_, err := db.Exec(
		"INSERT OR REPLACE INTO message_embeddings (message_id, workspace_id, embedding) VALUES (?, ?, ?)",
		messageID, workspaceID, embeddingJSON,
	)
	return err
}

func vectorToJSON(v []float32) string {
	result := "["
	for i, f := range v {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", f)
	}
	result += "]"
	return result
}

func SearchBySimilarity(db *DB, workspaceID string, embedding []float32, limit int) ([]SearchResult, error) {
	embeddingJSON := vectorToJSON(embedding)
	rows, err := db.Query(`
		SELECT m.id, m.workspace_id, m.content, m.timestamp, distance
		FROM message_embeddings
		WHERE message_embeddings MATCH ? AND workspace_id = ?
		ORDER BY distance
		LIMIT ?`,
		embeddingJSON, workspaceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.MessageID, &r.WorkspaceID, &r.Content, &r.Timestamp, &r.Similarity); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func SearchByKeyword(db *DB, workspaceID string, query string, limit int) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT m.id, m.workspace_id, m.content, m.timestamp, 0 as similarity
		FROM messages_fts fts
		JOIN messages m ON m.id = fts.rowid
		WHERE messages_fts MATCH ? AND fts.workspace_id = ?
		ORDER BY rank
		LIMIT ?`,
		query, workspaceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.MessageID, &r.WorkspaceID, &r.Content, &r.Timestamp, &r.Similarity); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
