package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/textclaw/textclaw/pkg/socket"
)

func NotifyCmd() *cobra.Command {
	var urgent bool
	var workspace string
	var suppress bool

	cmd := &cobra.Command{
		Use:   "notify [message]",
		Short: "Send a notification message",
		Long:  "Send a message to the user via the running daemon",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notifySend(args[0], urgent, workspace, suppress)
		},
	}

	cmd.Flags().BoolVarP(&urgent, "urgent", "u", false, "Send as urgent message")
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace ID (overrides config)")
	cmd.Flags().BoolVarP(&suppress, "suppress", "s", false, "Suppress if message contains 'No updates' (heartbeat mode)")

	return cmd
}

type WorkspaceConfig struct {
	Workspace string           `json:"workspace"`
	Channel   string           `json:"channel"`
	Target    string           `json:"target"`
	Heartbeat *HeartbeatConfig `json:"heartbeat,omitempty"`
}

type HeartbeatConfig struct {
	Enabled bool   `json:"enabled"`
	Every   string `json:"every"`
}

func loadWorkspaceConfig() (*WorkspaceConfig, error) {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".textclaw.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg WorkspaceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func notifySend(message string, urgent bool, workspace string, suppress bool) error {
	homeDir, _ := os.UserHomeDir()

	if workspace == "" {
		cfg, err := loadWorkspaceConfig()
		if err != nil {
			return fmt.Errorf("failed to load workspace config: %w", err)
		}
		workspace = cfg.Workspace
	}

	workspacePath := filepath.Join(homeDir, ".textclaw", "workspaces", workspace)
	wsConfig, err := loadWorkspaceConfigFile(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to load workspace config: %w", err)
	}

	if wsConfig == nil {
		return fmt.Errorf("workspace %s not found", workspace)
	}

	if suppress && message == "No updates" {
		return nil
	}

	socketPath := filepath.Join(homeDir, ".textclaw", "textclaw.sock")

	client := socket.NewClient(socketPath)

	if err := client.SendNotify(workspace, message, wsConfig.Target, urgent); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	fmt.Println("Message sent")
	return nil
}

func loadWorkspaceConfigFile(workspacePath string) (*WorkspaceConfig, error) {
	configPath := filepath.Join(workspacePath, ".textclaw.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg WorkspaceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
