package commands

import (
	"github.com/Martins6/textclaw/internal/database"
)

type Registry struct {
	db *database.DB
}

func NewRegistry(db *database.DB) *Registry {
	return &Registry{db: db}
}

func (r *Registry) GetCommands() ([]database.Command, error) {
	return database.GetCommands(r.db)
}

func (r *Registry) GetCommandByName(name string) (*database.Command, error) {
	return database.GetCommandByName(r.db, name)
}

func (r *Registry) SeedDefaultCommands() error {
	defaultCommands := []struct {
		name        string
		description string
		action      string
	}{
		{"new", "Start a new session", "new_session"},
		{"help", "Show available commands", "show_help"},
		{"status", "Check workspace status", "show_status"},
	}

	for _, cmd := range defaultCommands {
		if err := database.InsertCommand(r.db, cmd.name, cmd.description, cmd.action); err != nil {
			return err
		}
	}

	return nil
}
