package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Martins6/textclaw/internal/cli"
)

var rootCmd = &cobra.Command{
	Use:   "textclaw",
	Short: "TextClaw CLI - AI-powered messaging daemon",
	Long:  `TextClaw connects to messaging channels (Telegram, WhatsApp) and routes messages to AI agent containers.`,
}

func init() {
	rootCmd.AddCommand(cli.InitCmd())
	rootCmd.AddCommand(cli.ConfigCmd())
	rootCmd.AddCommand(cli.DaemonCmd())
	rootCmd.AddCommand(cli.NotifyCmd())
	rootCmd.AddCommand(cli.ContextCmd())
	rootCmd.AddCommand(cli.OpenCodeCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
