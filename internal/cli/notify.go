package cli

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/spf13/cobra"
)

func NotifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "notify [message]",
		Short: "Send a notification message",
		Long:  "Send a message to the user via the running daemon",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return notifySend(args[0])
		},
	}
}

type NotifyRequest struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type NotifyResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func notifySend(message string) error {
	conn, err := net.Dial("unix", textclawSocket)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	req := NotifyRequest{
		Type:    "notify",
		Message: message,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := conn.Write(reqData); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var resp NotifyResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("daemon error: %s", resp.Error)
	}

	fmt.Println("Message sent")
	return nil
}
