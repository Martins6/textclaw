package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Martins6/textclaw/internal/container"
	"github.com/Martins6/textclaw/internal/database"
)

type Runner struct {
	containerMgr      *container.Manager
	workspaceDir      string
	openCodeConfigDir string
	openCodeAuthDir   string
	image             string
	openCodePort      string
	currentSessions   map[string]string
	workspacePorts    map[string]string
	workspaceIPs      map[string]string
	mu                sync.RWMutex
	db                *database.DB
}

type RunnerOption func(*Runner)

func WithImage(image string) RunnerOption {
	return func(r *Runner) {
		r.image = image
	}
}

func WithOpenCodePort(port string) RunnerOption {
	return func(r *Runner) {
		r.openCodePort = port
	}
}

func New(workspaceDir, openCodeConfigDir, openCodeAuthDir string, db *database.DB, opts ...RunnerOption) (*Runner, error) {
	containerMgr, err := container.NewManager()
	if err != nil {
		return nil, err
	}

	r := &Runner{
		containerMgr:      containerMgr,
		workspaceDir:      workspaceDir,
		openCodeConfigDir: openCodeConfigDir,
		openCodeAuthDir:   openCodeAuthDir,
		image:             "opencode:latest",
		openCodePort:      "8080",
		currentSessions:   make(map[string]string),
		workspacePorts:    make(map[string]string),
		workspaceIPs:      make(map[string]string),
		db:                db,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.db != nil {
		if err := r.loadSessions(); err != nil {
			return nil, fmt.Errorf("failed to load sessions: %w", err)
		}
	}

	return r, nil
}

func (r *Runner) Close() error {
	return r.containerMgr.Close()
}

func (r *Runner) loadSessions() error {
	workspaces, err := r.getAllWorkspaces()
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		if ws.SessionID != nil && *ws.SessionID != "" {
			r.currentSessions[ws.ID] = *ws.SessionID
		}
	}
	return nil
}

func (r *Runner) getAllWorkspaces() ([]database.Workspace, error) {
	rows, err := r.db.Query("SELECT id, session_id FROM workspaces WHERE session_id IS NOT NULL AND session_id != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []database.Workspace
	for rows.Next() {
		var w database.Workspace
		if err := rows.Scan(&w.ID, &w.SessionID); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

func (r *Runner) GetAllWorkspaces() ([]database.Workspace, error) {
	rows, err := r.db.Query("SELECT id, session_id FROM workspaces")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []database.Workspace
	for rows.Next() {
		var w database.Workspace
		if err := rows.Scan(&w.ID, &w.SessionID); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

func (r *Runner) StartAllContainers(ctx context.Context) error {
	workspaces, err := r.GetAllWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to get workspaces: %w", err)
	}

	for _, ws := range workspaces {
		if err := r.EnsureContainer(ctx, ws.ID); err != nil {
			log.Printf("Failed to start container for workspace %s: %v", ws.ID, err)
			continue
		}
		log.Printf("Container started for workspace %s", ws.ID)
	}

	return nil
}

func (r *Runner) GetCurrentSession(workspaceID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.currentSessions[workspaceID]
}

func (r *Runner) SetCurrentSession(workspaceID, sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentSessions[workspaceID] = sessionID

	if r.db != nil {
		if err := database.UpdateWorkspaceSession(r.db, workspaceID, sessionID); err != nil {
			log.Printf("Failed to persist session for workspace %s: %v", workspaceID, err)
		}
	}
}

type Message struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type Response struct {
	Info  Info   `json:"info"`
	Parts []Part `json:"parts"`
}

type Info struct {
	SessionID string `json:"session_id"`
}

var ErrInvalidSession = fmt.Errorf("invalid session")

type ExecuteResult struct {
	Response         string
	SessionRecreated bool
}

func (r *Runner) Execute(ctx context.Context, workspaceID, prompt string) (ExecuteResult, error) {
	if err := r.EnsureContainer(ctx, workspaceID); err != nil {
		return ExecuteResult{}, fmt.Errorf("failed to ensure container: %w", err)
	}

	ip := "localhost"

	sessionID := r.GetCurrentSession(workspaceID)
	if sessionID == "" {
		var err error
		sessionID, err = r.ensureSession(ctx, ip, workspaceID)
		if err != nil {
			return ExecuteResult{}, fmt.Errorf("failed to ensure session: %w", err)
		}
		r.SetCurrentSession(workspaceID, sessionID)
	}

	response, err := r.sendMessage(ctx, ip, workspaceID, sessionID, prompt)
	if err != nil {
		if err == ErrInvalidSession {
			log.Printf("Session expired, creating new session for workspace %s", workspaceID)
			r.SetCurrentSession(workspaceID, "")

			sessionID, err = r.ensureSession(ctx, ip, workspaceID)
			if err != nil {
				return ExecuteResult{}, fmt.Errorf("failed to create new session: %w", err)
			}
			r.SetCurrentSession(workspaceID, sessionID)

			response, err = r.sendMessage(ctx, ip, workspaceID, sessionID, prompt)
			if err != nil {
				return ExecuteResult{}, fmt.Errorf("failed to send message: %w", err)
			}

			return ExecuteResult{
				Response:         r.formatResponse(response),
				SessionRecreated: true,
			}, nil
		}
		return ExecuteResult{}, fmt.Errorf("failed to send message: %w", err)
	}

	return ExecuteResult{Response: r.formatResponse(response)}, nil
}

func (r *Runner) EnsureContainer(ctx context.Context, workspaceID string) error {
	containerName := fmt.Sprintf("textclaw-%s", workspaceID)

	exists, running, containerID, err := r.containerMgr.ContainerExists(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container: %w", err)
	}

	if !exists {
		_, port, ip, err := r.createAndStartContainer(ctx, workspaceID, containerName)
		if err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
		r.mu.Lock()
		r.workspacePorts[workspaceID] = port
		r.workspaceIPs[workspaceID] = ip
		r.mu.Unlock()
		return nil
	}

	if !running {
		err = r.containerMgr.RemoveContainer(ctx, containerID, true)
		if err != nil {
			return fmt.Errorf("failed to remove stopped container: %w", err)
		}
		_, port, ip, err := r.createAndStartContainer(ctx, workspaceID, containerName)
		if err != nil {
			return fmt.Errorf("failed to recreate container: %w", err)
		}
		r.mu.Lock()
		r.workspacePorts[workspaceID] = port
		r.workspaceIPs[workspaceID] = ip
		r.mu.Unlock()
		return nil
	}

	port, err := r.containerMgr.GetContainerPort(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container port: %w", err)
	}
	ip, err := r.containerMgr.GetContainerIP(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container IP: %w", err)
	}
	r.mu.Lock()
	r.workspacePorts[workspaceID] = port
	r.workspaceIPs[workspaceID] = ip
	r.mu.Unlock()

	log.Printf("Using existing container for workspace %s on port %s", workspaceID, port)

	err = r.containerMgr.WaitForPort(ctx, containerID, port, 120*time.Second)
	if err != nil {
		return fmt.Errorf("container running but OpenCode server not ready: %w", err)
	}

	return nil
}

func (r *Runner) createAndStartContainer(ctx context.Context, workspaceID, containerName string) (string, string, string, error) {
	workspacePath := r.getWorkspacePath(workspaceID)

	if !r.containerMgr.ImageExists(r.image) {
		if err := r.containerMgr.PullImage(ctx, r.image); err != nil {
			return "", "", "", fmt.Errorf("failed to pull image: %w", err)
		}
	}

	cfg := container.ContainerConfig{
		Image:             r.image,
		Name:              containerName,
		WorkspaceDir:      workspacePath,
		OpenCodeConfigDir: r.openCodeConfigDir,
		OpenCodeAuthDir:   r.openCodeAuthDir,
		OpenCodeStateDir:  r.getWorkspaceStatePath(workspaceID),
	}

	containerID, err := r.containerMgr.CreateContainer(ctx, cfg)
	if err != nil {
		return "", "", "", err
	}

	err = r.containerMgr.StartContainer(ctx, containerID)
	if err != nil {
		return "", "", "", err
	}

	port, err := r.containerMgr.GetContainerPort(ctx, containerID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get container port: %w", err)
	}

	err = r.containerMgr.WaitForPort(ctx, containerID, port, 120*time.Second)
	if err != nil {
		return "", "", "", fmt.Errorf("container started but OpenCode server not ready: %w", err)
	}

	ip, err := r.containerMgr.GetContainerIP(ctx, containerID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get container IP: %w", err)
	}

	log.Printf("Container %s started on port %s, IP %s", containerID[:12], port, ip)

	return containerID, port, ip, nil
}

func (r *Runner) getWorkspacePath(workspaceID string) string {
	return fmt.Sprintf("%s/%s", r.workspaceDir, workspaceID)
}

func (r *Runner) getWorkspaceStatePath(workspaceID string) string {
	return fmt.Sprintf("%s/%s/opencode-state", r.workspaceDir, workspaceID)
}

func (r *Runner) getWorkspacePort(workspaceID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.workspacePorts[workspaceID]
}

func (r *Runner) getWorkspaceIP(workspaceID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.workspaceIPs[workspaceID]
}

func (r *Runner) ensureSession(ctx context.Context, ip, workspaceID string) (string, error) {
	port := r.getWorkspacePort(workspaceID)
	if port == "" {
		port = r.openCodePort
	}
	url := fmt.Sprintf("http://%s:%s/session", ip, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create session: %s - %s", resp.Status, string(bodyBytes))
	}

	var sessionResp struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to decode session response: %w, body: %s", err, string(bodyBytes))
	}

	if sessionResp.ID == "" {
		return "", fmt.Errorf("empty session ID returned from OpenCode server")
	}

	return sessionResp.ID, nil
}

func (r *Runner) sendMessage(ctx context.Context, ip, workspaceID, sessionID, prompt string) (*Response, error) {
	port := r.getWorkspacePort(workspaceID)
	if port == "" {
		port = r.openCodePort
	}
	url := fmt.Sprintf("http://%s:%s/session/%s/message", ip, port, sessionID)

	msg := Message{
		Parts: []Part{
			{Type: "text", Text: prompt},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to send message: %s - %s", resp.Status, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(bodyBytes) > 0 && bodyBytes[0] == '<' {
		maxLen := 200
		if len(bodyBytes) > maxLen {
			bodyBytes = bodyBytes[:maxLen]
		}
		log.Printf("Invalid session response from OpenCode (sessionID: %s), body: %s", sessionID, string(bodyBytes))
		return nil, ErrInvalidSession
	}

	var response Response
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (r *Runner) formatResponse(resp *Response) string {
	var result string
	for _, part := range resp.Parts {
		if part.Type == "text" {
			result += part.Text
		}
	}
	return result
}

func (r *Runner) NewSession(ctx context.Context, workspaceID string) (string, error) {
	containerName := fmt.Sprintf("textclaw-%s", workspaceID)

	exists, _, containerID, err := r.containerMgr.ContainerExists(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to check container: %w", err)
	}

	if !exists {
		return "", fmt.Errorf("container not running for workspace %s", workspaceID)
	}

	ip, err := r.containerMgr.GetContainerIP(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to get container IP: %w", err)
	}

	return r.ensureSession(ctx, ip, workspaceID)
}
