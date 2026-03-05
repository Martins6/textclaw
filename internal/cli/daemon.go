package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Martins6/textclaw/internal/daemon"
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

	cmd.AddCommand(&cobra.Command{
		Use:    "run",
		Short:  "Run daemon (internal)",
		RunE:   daemonRun,
		Hidden: true,
	})

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "View daemon logs",
		RunE:  daemonLogs,
	}
	logsCmd.Flags().StringP("date", "d", time.Now().Format("2006-01-02"), "Date for log file (YYYY-MM-DD)")
	logsCmd.Flags().IntP("lines", "n", 100, "Number of lines to show")
	logsCmd.Flags().BoolP("tail", "f", false, "Follow log output")
	cmd.AddCommand(logsCmd)

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

	proc := exec.Command(daemonPath, "daemon", "run")
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

func daemonRun(cmd *cobra.Command, args []string) error {
	return daemon.Run()
}

func daemonLogs(cmd *cobra.Command, args []string) error {
	workspaceID := "main"
	if len(args) > 0 {
		workspaceID = args[0]
	}

	dateStr, _ := cmd.Flags().GetString("date")
	lines, _ := cmd.Flags().GetInt("lines")
	tail, _ := cmd.Flags().GetBool("tail")

	logFile := filepath.Join(textclawDir, "logs", workspaceID, dateStr+".log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s", logFile)
	}

	if tail {
		return tailLog(logFile, lines)
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	linesArr := splitLines(string(content))
	start := 0
	if len(linesArr) > lines {
		start = len(linesArr) - lines
	}

	for i := start; i < len(linesArr); i++ {
		fmt.Println(linesArr[i])
	}

	return nil
}

func splitLines(s string) []string {
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func tailLog(logFile string, initialLines int) error {
	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	lastSize := fileInfo.Size()

	content, err := os.ReadFile(logFile)
	if err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	linesArr := splitLines(string(content))
	start := 0
	if len(linesArr) > initialLines {
		start = len(linesArr) - initialLines
	}

	for i := start; i < len(linesArr); i++ {
		fmt.Println(linesArr[i])
	}

	for {
		time.Sleep(1 * time.Second)

		fileInfo, err := os.Stat(logFile)
		if err != nil {
			continue
		}

		newSize := fileInfo.Size()
		if newSize > lastSize {
			content, err := os.ReadFile(logFile)
			if err != nil {
				continue
			}

			linesArr := splitLines(string(content))
			if len(linesArr) > 0 {
				fmt.Println(linesArr[len(linesArr)-1])
			}

			lastSize = newSize
		}
	}
}
