package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type WorkspaceConfigFile struct {
	Heartbeat *HeartbeatConfig `json:"heartbeat,omitempty"`
	Agent     *AgentConfig     `json:"agent,omitempty"`
}

type HeartbeatConfig struct {
	Enabled  bool     `json:"enabled"`
	Schedule string   `json:"schedule"`
	NotifyOn []string `json:"notify_on"`
}

type AgentConfig struct {
	ReadHeartbeats bool `json:"read_heartbeats"`
}

func LoadWorkspaceConfig(workspacePath string) (*WorkspaceConfigFile, error) {
	configPath := filepath.Join(workspacePath, ".textclaw.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read workspace config: %w", err)
	}

	var cfg WorkspaceConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse workspace config: %w", err)
	}

	return &cfg, nil
}

func SaveWorkspaceConfig(workspacePath string, cfg *WorkspaceConfigFile) error {
	configPath := filepath.Join(workspacePath, ".textclaw.json")

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace config: %w", err)
	}

	return nil
}
