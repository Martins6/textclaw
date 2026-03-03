package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Martins6/textclaw/internal/config"
)

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage TextClaw configuration",
		Long:  "Get or set configuration values in setup.toml",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get [key]",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configGet(args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configSet(args[0], args[1])
		},
	})

	return cmd
}

func configPath() string {
	return filepath.Join(textclawDir, "setup.toml")
}

func configGet(key string) error {
	cfg, err := config.Load(configPath())
	if err != nil {
		return err
	}

	switch key {
	case "container.image":
		fmt.Println(cfg.Container.Image)
	case "container.volumes":
		for _, v := range cfg.Container.Volumes {
			fmt.Println(v)
		}
	case "workspace.base_path":
		fmt.Println(cfg.Workspace.BasePath)
	case "telegram.token":
		fmt.Println(cfg.Telegram.Token)
	case "telegram.allowed_users":
		for _, u := range cfg.Telegram.AllowedUsers {
			fmt.Println(u)
		}
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}

func configSet(key, value string) error {
	cfg, err := config.Load(configPath())
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cfg = &config.Config{}
	}

	switch key {
	case "container.image":
		cfg.Container.Image = value
	case "workspace.base_path":
		cfg.Workspace.BasePath = value
	case "telegram.token":
		cfg.Telegram.Token = value
	case "telegram.allowed_users":
		cfg.Telegram.AllowedUsers = []string{value}
	default:
		return fmt.Errorf("unknown key: %s", key)
	}

	return config.Save(configPath(), cfg)
}
