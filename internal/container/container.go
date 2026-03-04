package container

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type Manager struct {
	cli *docker.Client
}

func NewManager() (*Manager, error) {
	var cli *docker.Client
	var err error

	dockerSocket := getDockerSocket()
	if dockerSocket != "" {
		cli, err = docker.NewClient(dockerSocket)
	} else {
		cli, err = docker.NewClientFromEnv()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &Manager{cli: cli}, nil
}

func getDockerSocket() string {
	if runtime.GOOS == "darwin" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			socketPath := filepath.Join(homeDir, ".docker", "run", "docker.sock")
			if _, err := os.Stat(socketPath); err == nil {
				return "unix://" + socketPath
			}
		}
	}
	return ""
}

func (m *Manager) Close() error {
	return nil
}

func (m *Manager) PullImage(ctx context.Context, imageName string) error {
	err := m.cli.PullImage(docker.PullImageOptions{
		Repository:   imageName,
		OutputStream: &NopWriter{},
	}, docker.AuthConfiguration{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	return nil
}

func (m *Manager) ImageExists(imageName string) bool {
	_, err := m.cli.InspectImage(imageName)
	return err == nil
}

func (m *Manager) BuildImage(ctx context.Context, imageName, dockerfilePath string) error {
	dir := filepath.Dir(dockerfilePath)

	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build image %s: %w\nOutput: %s", imageName, err, string(output))
	}
	return nil
}

type NopWriter struct{}

func (n *NopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type ContainerConfig struct {
	Image             string
	Name              string
	WorkspaceDir      string
	OpenCodeConfigDir string
	OpenCodeAuthDir   string
	OpenCodeStateDir  string
	MainWorkspace     bool
	OtherWorkspaces   []string
}

func (m *Manager) CreateContainer(ctx context.Context, cfg ContainerConfig) (string, error) {
	binds := []string{
		fmt.Sprintf("%s:/workspace:rw", cfg.WorkspaceDir),
	}

	if cfg.OpenCodeConfigDir != "" {
		configDirPath := cfg.OpenCodeConfigDir
		binds = append(binds, fmt.Sprintf("%s:/home/user/.config/opencode:ro", configDirPath))
	}

	if cfg.OpenCodeStateDir != "" {
		statePath := cfg.OpenCodeStateDir
		if _, err := os.Stat(statePath); os.IsNotExist(err) {
			if err := os.MkdirAll(statePath, 0755); err == nil {
				binds = append(binds, fmt.Sprintf("%s:/home/user/.local/state:rw", statePath))
			}
		} else {
			binds = append(binds, fmt.Sprintf("%s:/home/user/.local/state:rw", statePath))
		}
	}

	if cfg.OpenCodeAuthDir != "" {
		authDirPath := cfg.OpenCodeAuthDir
		binds = append(binds, fmt.Sprintf("%s:/home/user/.local/share/opencode:rw", authDirPath))
	}

	if cfg.MainWorkspace {
		for _, ws := range cfg.OtherWorkspaces {
			binds = append(binds, fmt.Sprintf("%s:/home/user/workspaces/%s:ro", cfg.WorkspaceDir, ws))
		}
	}

	containerCfg := docker.Config{
		Image: cfg.Image,
		Env: []string{
			"OPENCODE_HOST=0.0.0.0",
			"OPENCODE_PORT=8080",
		},
		ExposedPorts: map[docker.Port]struct{}{
			"8080/tcp": {},
		},
	}

	hostCfg := docker.HostConfig{
		Binds: binds,
		PortBindings: map[docker.Port][]docker.PortBinding{
			"8080/tcp": {{HostIP: "0.0.0.0", HostPort: ""}},
		},
	}

	opts := docker.CreateContainerOptions{
		Name:       cfg.Name,
		Config:     &containerCfg,
		HostConfig: &hostCfg,
	}

	resp, err := m.cli.CreateContainer(opts)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

func (m *Manager) StartContainer(ctx context.Context, containerID string) error {
	return m.cli.StartContainer(containerID, nil)
}

func (m *Manager) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	return m.cli.StopContainer(containerID, uint(timeout.Seconds()))
}

func (m *Manager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return m.cli.RemoveContainer(docker.RemoveContainerOptions{
		ID:    containerID,
		Force: force,
	})
}

func (m *Manager) ContainerExists(ctx context.Context, name string) (exists bool, running bool, containerID string, err error) {
	container, err := m.cli.InspectContainer(name)
	if err != nil {
		if _, ok := err.(*docker.NoSuchContainer); ok {
			return false, false, "", nil
		}
		return false, false, "", err
	}
	return true, container.State.Running, container.ID, nil
}

func (m *Manager) GetContainerPort(ctx context.Context, containerID string) (string, error) {
	inspect, err := m.cli.InspectContainer(containerID)
	if err != nil {
		return "", err
	}

	if inspect.NetworkSettings == nil {
		return "", fmt.Errorf("no network settings")
	}

	for _, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) > 0 {
			return bindings[0].HostPort, nil
		}
	}

	return "", fmt.Errorf("no exposed ports found")
}

func (m *Manager) WaitForPort(ctx context.Context, containerID string, port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		url := fmt.Sprintf("http://localhost:%s/global/health", port)
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	return fmt.Errorf("timeout waiting for port %s", port)
}

func (m *Manager) GetContainerIP(ctx context.Context, containerID string) (string, error) {
	inspect, err := m.cli.InspectContainer(containerID)
	if err != nil {
		return "", err
	}

	if inspect.NetworkSettings == nil {
		return "", fmt.Errorf("no network settings")
	}

	for _, network := range inspect.NetworkSettings.Networks {
		return network.IPAddress, nil
	}

	return "", fmt.Errorf("no network IP found")
}
