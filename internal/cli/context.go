package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/textclaw/textclaw/pkg/socket"
)

func ContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Search conversation history",
		Long:  "Search historical messages in the workspace using semantic, keyword, or recent search",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "similar [query]",
		Short: "Semantic search - find messages similar to query",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return contextSearch("semantic", args[0], 10)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "search [query]",
		Short: "Keyword search - find messages containing term",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return contextSearch("keyword", args[0], 10)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "find [query]",
		Short: "Full-text search - find messages containing term",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return contextSearch("find", args[0], 10)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "recent [limit]",
		Short: "Get recent messages",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit := 10
			if len(args) > 0 {
				fmt.Sscanf(args[0], "%d", &limit)
			}
			return contextSearch("recent", "", limit)
		},
	})

	return cmd
}

type ContextResult struct {
	MessageID   int64     `json:"message_id"`
	WorkspaceID string    `json:"workspace_id"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	Similarity  float64   `json:"similarity"`
}

func contextSearch(searchType, query string, limit int) error {
	cfg, err := loadContextWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load workspace config: %w", err)
	}

	if cfg == nil || cfg.Workspace == "" {
		return fmt.Errorf("no workspace configured. Run 'textclaw init' or set workspace in ~/.textclaw.json")
	}

	socketPath := filepath.Join(textclawDir, "textclaw.sock")
	client := socket.NewClient(socketPath)

	results, err := client.ContextSearch(cfg.Workspace, searchType, query, limit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	fmt.Printf("Found %d result(s):\n\n", len(results))
	for i, r := range results {
		fmt.Printf("%d. [%s] %s\n", i+1, r.Timestamp.Format("2006-01-02 15:04"), r.Content)
		if r.Similarity > 0 {
			fmt.Printf("   Similarity: %.4f\n", r.Similarity)
		}
		fmt.Println()
	}

	return nil
}

func loadContextWorkspaceConfig() (*WorkspaceConfig, error) {
	configPath := filepath.Join(textclawDir, "textclaw.json")

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
