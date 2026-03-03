package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const migrationVersion = "001_initial"

type Contact struct {
	ID          string
	WorkspaceID string
	Role        string
	CreatedAt   time.Time
}

type Workspace struct {
	ID          string
	ContainerID *string
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

	db, err := sql.Open("sqlite3", absPath+"?cache=shared&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(1)

	return &DB{DB: db, path: absPath}, nil
}

func RunMigrations(db *DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migrationVersion).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check migration: %w", err)
	}

	if count > 0 {
		return nil
	}

	migrationPath := filepath.Join("internal", "database", "migrations", migrationVersion+".sql")
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migrationVersion)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
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
		"SELECT id, container_id, created_at FROM workspaces WHERE id = ?",
		id,
	).Scan(&w.ID, &w.ContainerID, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
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
