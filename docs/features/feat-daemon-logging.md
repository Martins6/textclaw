# Overview

Comprehensive logging system that writes all daemon events and workspace interactions to log files, with CLI access to view and tail logs.

# Details

- Writes daemon events to ~/.textclaw/logs/{workspace_id}/{YYYY-MM-DD}.log
- Logs include INPUT (incoming messages), OUTPUT (outgoing messages), DAEMON (routing decisions), EXECUTE (AI prompt execution), CONTAINER (container lifecycle)
- Thread-safe log writing with daily log file rotation
- CLI command `textclaw daemon logs <workspace> [--date YYYY-MM-DD] [--lines N] [--tail/-f]` to retrieve logs
- Supports live tail with -f flag
- Integrates with all daemon components: daemon.go, runner.go, telegram.go

# File Paths

- internal/daemon/logs/log_manager.go
- internal/cli/daemon.go (daemon logs subcommand)
- internal/daemon/daemon.go (channelIn, channelOut, daemonLog functions)
- internal/daemon/runner/runner.go (container and execution logging)
