package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

func DaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage TextClaw daemon",
		Long:  "Start or stop the TextClaw daemon background process",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		RunE:  daemonStart,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		RunE:  daemonStop,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		RunE:  daemonStatus,
	})

	return cmd
}

func pidFile() string {
	return filepath.Join(textclawDir, "textclaw.pid")
}

func daemonStart(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(pidFile()); err == nil {
		pid, err := os.ReadFile(pidFile())
		if err == nil {
			if isProcessRunning(string(pid)) {
				return fmt.Errorf("daemon is already running (PID: %s)", string(pid))
			}
		}
		os.Remove(pidFile())
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find executable: %w", err)
	}

	daemonPath := filepath.Join(textclawDir, "textclaw", "daemon")
	if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
		daemonPath = execPath
	}

	daemonBin := filepath.Join(textclawDir, "bin", "textclaw-daemon")
	if _, err := os.Stat(daemonBin); err == nil {
		daemonPath = daemonBin
	}

	proc := exec.Command(daemonPath, "daemon")
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	proc.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := proc.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	if err := os.WriteFile(pidFile(), []byte(strconv.Itoa(proc.Process.Pid)), 0644); err != nil {
		proc.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	fmt.Printf("Daemon started (PID: %d)\n", proc.Process.Pid)
	return nil
}

func daemonStop(cmd *cobra.Command, args []string) error {
	pidBytes, err := os.ReadFile(pidFile())
	if err != nil {
		return fmt.Errorf("daemon is not running (no PID file)")
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		return fmt.Errorf("invalid PID file")
	}

	if !isProcessRunning(string(pidBytes)) {
		os.Remove(pidFile())
		return fmt.Errorf("daemon is not running")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := proc.Kill(); err != nil {
		return fmt.Errorf("failed to kill daemon: %w", err)
	}

	os.Remove(pidFile())
	fmt.Println("Daemon stopped")
	return nil
}

func daemonStatus(cmd *cobra.Command, args []string) error {
	pidBytes, err := os.ReadFile(pidFile())
	if err != nil {
		fmt.Println("Daemon is not running")
		return nil
	}

	if isProcessRunning(string(pidBytes)) {
		fmt.Printf("Daemon is running (PID: %s)\n", string(pidBytes))
		return nil
	}

	fmt.Printf("Daemon is not running (stale PID file)\n")
	os.Remove(pidFile())
	return nil
}

func isProcessRunning(pidStr string) bool {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
