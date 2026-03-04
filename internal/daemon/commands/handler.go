package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Martins6/textclaw/internal/daemon/listener"
	"github.com/Martins6/textclaw/internal/daemon/runner"
)

type Handler struct {
	registry *Registry
	runner   *runner.Runner
	adapter  listener.Adapter
}

func NewHandler(registry *Registry, runner *runner.Runner, adapter listener.Adapter) *Handler {
	return &Handler{
		registry: registry,
		runner:   runner,
		adapter:  adapter,
	}
}

func (h *Handler) HandleCommand(ctx context.Context, msg listener.Message, workspaceID string) (bool, error) {
	content := strings.TrimSpace(msg.Content)

	if !strings.HasPrefix(content, "/") {
		return false, nil
	}

	parts := strings.Fields(content)
	commandName := strings.TrimPrefix(parts[0], "/")

	if commandName == "new" || strings.HasPrefix(commandName, "new ") {
		return h.handleNewSession(ctx, msg, workspaceID)
	}

	cmd, err := h.registry.GetCommandByName(commandName)
	if err != nil {
		if strings.HasPrefix(content, "/") {
			return true, h.adapter.Send(msg.ChatID, "Unknown command. Use /help to see available commands.")
		}
		return false, nil
	}

	switch cmd.Action {
	case "show_help":
		return h.handleHelp(msg)
	case "show_status":
		return h.handleStatus(msg, workspaceID)
	case "new_session":
		return h.handleNewSession(ctx, msg, workspaceID)
	default:
		return true, h.adapter.Send(msg.ChatID, "Command not implemented: "+cmd.Name)
	}
}

func (h *Handler) handleHelp(msg listener.Message) (bool, error) {
	cmds, err := h.registry.GetCommands()
	if err != nil {
		return true, h.adapter.Send(msg.ChatID, "Error loading commands: "+err.Error())
	}

	var helpText strings.Builder
	helpText.WriteString("Available commands:\n")
	for _, cmd := range cmds {
		helpText.WriteString(fmt.Sprintf("/%s - %s\n", cmd.Name, cmd.Description))
	}

	return true, h.adapter.Send(msg.ChatID, helpText.String())
}

func (h *Handler) handleStatus(msg listener.Message, workspaceID string) (bool, error) {
	sessionID := h.runner.GetCurrentSession(workspaceID)

	var status strings.Builder
	status.WriteString(fmt.Sprintf("Workspace: %s\n", workspaceID))
	if sessionID != "" {
		status.WriteString(fmt.Sprintf("Session: active (%s)\n", sessionID))
	} else {
		status.WriteString("Session: inactive\n")
	}
	status.WriteString("Container: running")

	return true, h.adapter.Send(msg.ChatID, status.String())
}

func (h *Handler) handleNewSession(ctx context.Context, msg listener.Message, workspaceID string) (bool, error) {
	sessionID, err := h.runner.NewSession(ctx, workspaceID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create new session: %v", err)
		return true, h.adapter.Send(msg.ChatID, errMsg)
	}

	h.runner.SetCurrentSession(workspaceID, sessionID)

	cmds, _ := h.registry.GetCommands()
	var commandList []string
	for _, cmd := range cmds {
		commandList = append(commandList, "/"+cmd.Name)
	}

	response := fmt.Sprintf("New session created! Previous context cleared.\n\nAvailable commands: %s", strings.Join(commandList, ", "))

	return true, h.adapter.Send(msg.ChatID, response)
}
