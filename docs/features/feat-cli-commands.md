# Overview

Phase 2 implements three CLI commands for TextClaw: config management, daemon lifecycle control, and message notification sending via Unix socket communication.

# Details

- Register CLI subcommands in main.go using Cobra framework
- Implement config get/set subcommands for setup.toml values
- Implement daemon start/stop subcommands using PID file pattern
- Implement notify command that communicates with daemon via Unix socket
- Handle daemon background process management
- Define Unix socket path for CLI to daemon communication

# File Paths

- cmd/textclaw/main.go
- internal/cli/config.go
- internal/cli/daemon.go
- internal/cli/notify.go
