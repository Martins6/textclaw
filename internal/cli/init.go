package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/textclaw/textclaw/internal/config"
	"github.com/textclaw/textclaw/internal/database"
)

func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize TextClaw",
		Long:  "Create ~/.textclaw/ structure with templates and database",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Init()
		},
	}
}

func Init() error {
	if _, err := os.Stat(textclawDir); err == nil {
		return fmt.Errorf("TextClaw is already initialized at %s", textclawDir)
	}

	dirs := []string{
		filepath.Join(textclawDir, "textclaw"),
		filepath.Join(textclawDir, ".opencode"),
		filepath.Join(textclawDir, "workspaces", "main"),
		filepath.Join(textclawDir, "database", "migrations"),
		filepath.Join(textclawDir, "heartbeats"),
		filepath.Join(textclawDir, "cronjobs"),
		filepath.Join(textclawDir, "models"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	templateFiles := map[string]string{
		"templates/Dockerfile":          filepath.Join(textclawDir, "Dockerfile"),
		"templates/setup.toml":          filepath.Join(textclawDir, "setup.toml"),
		"templates/opencode_install.sh": filepath.Join(textclawDir, "opencode_install.sh"),
		"templates/setup_linux.sh":      filepath.Join(textclawDir, "setup_linux.sh"),
	}

	for src, dst := range templateFiles {
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", src, err)
		}
		if err := os.WriteFile(dst, data, 0755); err != nil {
			return fmt.Errorf("failed to write %s: %w", dst, err)
		}
	}

	opencodeFiles := []string{
		"opencode.json", "AGENTS.md", "SOUL.md", "TOOLS.md", "USER.md", "HEARTBEATS.md",
	}
	for _, file := range opencodeFiles {
		src := filepath.Join("templates/.opencode", file)
		dst := filepath.Join(textclawDir, ".opencode", file)
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", src, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dst, err)
		}
	}

	mainWorkspaceFiles := []string{"AGENTS.md", "SOUL.md", "TOOLS.md", "USER.md"}
	for _, file := range mainWorkspaceFiles {
		src := filepath.Join("templates/workspaces/main", file)
		dst := filepath.Join(textclawDir, "workspaces", "main", file)
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", src, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dst, err)
		}
	}

	dbPath := filepath.Join(textclawDir, "database", "sqlite.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := database.CreateWorkspace(db, "main"); err != nil {
		return fmt.Errorf("failed to create main workspace: %w", err)
	}

	cfg := &config.Config{
		Container: config.ContainerConfig{
			Image: "textclaw/agent:latest",
			Volumes: []string{
				"./workspaces/{workspace}:/home/{user}:rw",
				"~/.local/share/opencode:/home/{user}/.local/share/opencode:ro",
			},
		},
		Workspace: config.WorkspaceConfig{
			BasePath: "./workspaces",
		},
	}

	configPath := filepath.Join(textclawDir, "setup.toml")
	if err := config.Save(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("TextClaw initialized at %s\n", textclawDir)

	if err := downloadEmbeddingModel(); err != nil {
		fmt.Printf("Warning: Failed to download embedding model: %v\n", err)
		fmt.Printf("You can download it manually from: https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF\n")
		fmt.Printf("Place it at: %s\n", filepath.Join(textclawDir, "models", "nomic-embed-text-v1.5-Q8_0.gguf"))
	} else {
		fmt.Printf("Embedding model downloaded successfully\n")
	}

	return nil
}

const (
	embeddingModelURL  = "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf?download=true"
	embeddingModelFile = "nomic-embed-text-v1.5-Q4_K_M.gguf"
)

func downloadEmbeddingModel() error {
	modelPath := filepath.Join(textclawDir, "models", embeddingModelFile)

	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}

	fmt.Printf("Downloading embedding model (~274MB)...\n")
	fmt.Printf("This may take a few minutes depending on your connection...\n")

	resp, err := http.Get(embeddingModelURL)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(modelPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(modelPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func downloadEmbeddingModelFromHF() error {
	modelPath := filepath.Join(textclawDir, "models", embeddingModelFile)

	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}

	cmd := exec.Command("huggingface-cli", "download", "nomic-ai/nomic-embed-text-v1.5-GGUF", embeddingModelFile, "--local-dir", filepath.Join(textclawDir, "models"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to download with huggingface-cli: %w. Output: %s", err, string(output))
	}

	expectedPath := filepath.Join(textclawDir, "models", embeddingModelFile)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		files, _ := os.ReadDir(filepath.Join(textclawDir, "models"))
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "nomic-embed-text") && strings.HasSuffix(f.Name(), ".gguf") {
				os.Rename(filepath.Join(textclawDir, "models", f.Name()), expectedPath)
				break
			}
		}
	}

	return nil
}
