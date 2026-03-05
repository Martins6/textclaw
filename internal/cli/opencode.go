package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Martins6/textclaw/internal/config"
)

var opencodeAuthDir = filepath.Join(textclawDir, "opencode-auth")

func OpenCodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opencode",
		Short: "Manage TextClaw OpenCode",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "auth",
		Short: "Authenticate with OpenCode providers",
		Long:  `Run opencode auth login in a container to authenticate with AI providers. The auth will be saved to ~/.textclaw/opencode-auth/`,
		RunE:  opencodeAuth,
	})

	return cmd
}

func opencodeAuth(cmd *cobra.Command, args []string) error {
	os.MkdirAll(opencodeAuthDir, 0755)
	os.MkdirAll(filepath.Join(textclawDir, "opencode-state"), 0755)

	cfg, _ := config.Load(configPath())
	image := "textclaw/agent:latest"
	if cfg != nil && cfg.Container.Image != "" {
		image = cfg.Container.Image
	}

	stateDir := filepath.Join(textclawDir, "opencode-state")

	fmt.Println("Running: script -q /dev/null -- docker run -it --rm -v", opencodeAuthDir+":/home/user/.local/share/opencode:rw", "-v", stateDir+":/home/user/.local/state:rw", image, "/home/user/.opencode/bin/opencode auth login")
	fmt.Println()

	dockerCmd := exec.Command(
		"script", "-q", "/dev/null", "--", "docker", "run", "-it", "--rm",
		"-v", fmt.Sprintf("%s:/home/user/.local/share/opencode:rw", opencodeAuthDir),
		"-v", fmt.Sprintf("%s:/home/user/.local/state:rw", stateDir),
		"-e", "HOME=/home/user",
		image,
		"/home/user/.opencode/bin/opencode", "auth", "login",
	)
	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}
