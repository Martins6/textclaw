package provisioner

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Martins6/textclaw/internal/database"
)

type Provisioner struct {
	db                 *database.DB
	workspaceBase      string
	templatePath       string
	opencodeConfigPath string
	mainUserID         string
}

func New(db *database.DB, workspaceBase string, templatePath string, opencodeConfigPath string, mainUserID string) *Provisioner {
	return &Provisioner{
		db:                 db,
		workspaceBase:      workspaceBase,
		templatePath:       templatePath,
		opencodeConfigPath: opencodeConfigPath,
		mainUserID:         mainUserID,
	}
}

func (p *Provisioner) EnsureWorkspace(contactID string) (workspaceID string, err error) {
	existing, err := database.GetContact(p.db, contactID)
	if err == nil {
		return existing.WorkspaceID, nil
	}

	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check contact: %w", err)
	}

	mainID := strings.ToLower(p.mainUserID)
	contactIDLower := strings.ToLower(contactID)
	mainIDNoAt := strings.ReplaceAll(mainID, "@", "")
	contactIDNoAt := strings.ReplaceAll(contactIDLower, "@", "")
	isMainUser := mainIDNoAt == contactIDNoAt

	workspaceID = slugify(contactID)

	if isMainUser {
		workspaceID = ""
	}

	if err := p.createWorkspace(workspaceID); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	if workspaceID != "" {
		if err := database.CreateWorkspace(p.db, workspaceID); err != nil {
			return "", fmt.Errorf("failed to create workspace in database: %w", err)
		}
	}

	role := "user"
	if isMainGroup(contactID) || isMainUser {
		role = "main"
	}

	if err := database.CreateContact(p.db, contactID, workspaceID, role); err != nil {
		return "", fmt.Errorf("failed to create contact: %w", err)
	}

	return workspaceID, nil
}

func (p *Provisioner) createWorkspace(workspaceID string) error {
	workspacePath := p.workspaceBase

	if workspaceID != "" {
		workspacePath = filepath.Join(p.workspaceBase, workspaceID)
	}

	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return err
	}

	filesDir := filepath.Join(workspacePath, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return err
	}

	stateDir := filepath.Join(workspacePath, "opencode-state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}

	if p.opencodeConfigPath != "" {
		if err := p.copyOpenCodeConfig(workspacePath); err != nil {
			return err
		}
	}

	if p.templatePath != "" {
		if err := p.copyTemplate(workspaceID); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provisioner) copyTemplate(workspaceID string) error {
	templateFiles := []string{
		"AGENTS.md",
		"SOUL.md",
		"TOOLS.md",
		"USER.md",
		"HEARTBEATS.md",
		"CRONJOBS.md",
	}

	workspacePath := filepath.Join(p.workspaceBase, workspaceID)

	for _, file := range templateFiles {
		src := filepath.Join(p.templatePath, file)
		dst := filepath.Join(workspacePath, file)

		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", file, err)
			}
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return fmt.Errorf("failed to write template %s: %w", file, err)
			}
		}
	}

	return nil
}

func (p *Provisioner) copyOpenCodeConfig(workspacePath string) error {
	configFiles := []string{
		"opencode.json",
	}

	stateDir := filepath.Join(workspacePath, "opencode-state")

	for _, file := range configFiles {
		src := filepath.Join(p.opencodeConfigPath, file)
		dst := filepath.Join(stateDir, file)

		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("failed to read opencode config %s: %w", file, err)
			}
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return fmt.Errorf("failed to write opencode config %s: %w", file, err)
			}
		}
	}

	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "@", "")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, s)
	s = strings.Trim(s, "-")
	return s
}

func isMainGroup(contactID string) bool {
	return strings.HasPrefix(contactID, "main") || contactID == "main"
}
