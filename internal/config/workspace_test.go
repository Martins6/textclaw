package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkspaceConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".textclaw.json")

	testConfig := `{
		"heartbeat": {
			"enabled": true,
			"schedule": "*/5 * * * *",
			"notify_on": ["changes", "errors"]
		},
		"agent": {
			"read_heartbeats": true
		}
	}`

	if err := os.WriteFile(configPath, []byte(testConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadWorkspaceConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load workspace config: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected config to not be nil")
	}

	if cfg.Heartbeat == nil {
		t.Fatal("expected heartbeat config to not be nil")
	}

	if !cfg.Heartbeat.Enabled {
		t.Error("expected heartbeat to be enabled")
	}

	if cfg.Heartbeat.Schedule != "*/5 * * * *" {
		t.Errorf("expected schedule */5 * * * *, got %s", cfg.Heartbeat.Schedule)
	}

	if len(cfg.Heartbeat.NotifyOn) != 2 {
		t.Errorf("expected 2 notify_on items, got %d", len(cfg.Heartbeat.NotifyOn))
	}

	if cfg.Agent == nil || !cfg.Agent.ReadHeartbeats {
		t.Error("expected agent config to have read_heartbeats enabled")
	}
}

func TestLoadWorkspaceConfigNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := LoadWorkspaceConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load workspace config: %v", err)
	}

	if cfg != nil {
		t.Error("expected nil config when file doesn't exist")
	}
}

func TestSaveWorkspaceConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &WorkspaceConfigFile{
		Heartbeat: &HeartbeatConfig{
			Enabled:  true,
			Schedule: "*/10 * * * *",
			NotifyOn: []string{"summary"},
		},
		Agent: &AgentConfig{
			ReadHeartbeats: false,
		},
	}

	if err := SaveWorkspaceConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save workspace config: %v", err)
	}

	loadedCfg, err := LoadWorkspaceConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loadedCfg.Heartbeat.Schedule != "*/10 * * * *" {
		t.Errorf("expected schedule */10 * * * *, got %s", loadedCfg.Heartbeat.Schedule)
	}
}
