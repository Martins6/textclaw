package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	MaxAge = 30
)

type LogManager struct {
	mu      sync.RWMutex
	files   map[string]*os.File
	logDir  string
	dateStr string
}

var (
	defaultManager *LogManager
	once           sync.Once
)

func GetManager() *LogManager {
	once.Do(func() {
		var err error
		defaultManager, err = NewLogManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize log manager: %v\n", err)
			defaultManager = &LogManager{
				files:   make(map[string]*os.File),
				logDir:  filepath.Join(os.Getenv("HOME"), ".textclaw", "logs"),
				dateStr: time.Now().Format("2006-01-02"),
			}
		}
	})
	return defaultManager
}

func NewLogManager() (*LogManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, ".textclaw", "logs")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	m := &LogManager{
		files:   make(map[string]*os.File),
		logDir:  logDir,
		dateStr: time.Now().Format("2006-01-02"),
	}

	m.cleanup()

	return m, nil
}

func (m *LogManager) cleanup() {
	entries, err := os.ReadDir(m.logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -MaxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspaceDir := filepath.Join(m.logDir, entry.Name())
		workspaceEntries, err := os.ReadDir(workspaceDir)
		if err != nil {
			continue
		}

		for _, wsEntry := range workspaceEntries {
			if wsEntry.IsDir() {
				continue
			}

			info, err := wsEntry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(workspaceDir, wsEntry.Name()))
			}
		}
	}
}

func (m *LogManager) getLogFile(workspaceID string) (*os.File, error) {
	m.mu.RLock()
	if f, ok := m.files[workspaceID]; ok {
		currentDate := time.Now().Format("2006-01-02")
		if m.dateStr == currentDate {
			m.mu.RUnlock()
			return f, nil
		}
		f.Close()
		delete(m.files, workspaceID)
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if f, ok := m.files[workspaceID]; ok {
		return f, nil
	}

	workspaceDir := filepath.Join(m.logDir, workspaceID)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace log directory: %w", err)
	}

	currentDate := time.Now().Format("2006-01-02")
	logFile := filepath.Join(workspaceDir, currentDate+".log")

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	m.files[workspaceID] = f
	m.dateStr = currentDate

	return f, nil
}

func (m *LogManager) Log(workspaceID, level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	f, err := m.getLogFile(workspaceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get log file: %v\n", err)
		return
	}

	f.WriteString(entry)
}

func Log(workspaceID, level, message string) {
	GetManager().Log(workspaceID, level, message)
}

func GetLogPath(workspaceID string) string {
	manager := GetManager()
	currentDate := time.Now().Format("2006-01-02")
	return filepath.Join(manager.logDir, workspaceID, currentDate+".log")
}

func ReadLog(workspaceID, date string) ([]byte, error) {
	manager := GetManager()
	logFile := filepath.Join(manager.logDir, workspaceID, date+".log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("log file not found for workspace %s on %s", workspaceID, date)
	}

	return os.ReadFile(logFile)
}

func ListWorkspaces() ([]string, error) {
	manager := GetManager()

	entries, err := os.ReadDir(manager.logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	var workspaces []string
	for _, entry := range entries {
		if entry.IsDir() {
			workspaces = append(workspaces, entry.Name())
		}
	}

	return workspaces, nil
}

func Close() {
	if defaultManager != nil {
		defaultManager.mu.Lock()
		defer defaultManager.mu.Unlock()

		for _, f := range defaultManager.files {
			f.Close()
		}
		defaultManager.files = make(map[string]*os.File)
	}
}
