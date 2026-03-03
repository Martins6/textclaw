package container

import (
	"context"
	"fmt"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type Manager struct {
	cli *docker.Client
}

func NewManager() (*Manager, error) {
	cli, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &Manager{cli: cli}, nil
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

type NopWriter struct{}

func (n *NopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

type ContainerConfig struct {
	Image           string
	Name            string
	WorkspaceDir    string
	OpenCodeAuth    string
	MainWorkspace   bool
	OtherWorkspaces []string
}

func (m *Manager) CreateContainer(ctx context.Context, cfg ContainerConfig) (string, error) {
	binds := []string{
		fmt.Sprintf("%s:/workspace:rw", cfg.WorkspaceDir),
	}

	if cfg.OpenCodeAuth != "" {
		binds = append(binds, fmt.Sprintf("%s:/home/user/.local/share/opencode:ro", cfg.OpenCodeAuth))
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

func (m *Manager) ContainerExists(ctx context.Context, name string) (bool, string, error) {
	container, err := m.cli.InspectContainer(name)
	if err != nil {
		if _, ok := err.(*docker.NoSuchContainer); ok {
			return false, "", nil
		}
		return false, "", err
	}
	return true, container.ID, nil
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
	for time.Now().Before(deadline) {
		inspect, err := m.cli.InspectContainer(containerID)
		if err != nil {
			return err
		}

		if inspect.NetworkSettings != nil {
			for _, bindings := range inspect.NetworkSettings.Ports {
				if len(bindings) > 0 && bindings[0].HostPort == port {
					return nil
				}
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
