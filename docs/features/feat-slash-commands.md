# Overview

Slash command system allows Telegram users to interact with TextClaw using commands like /new, /help, and /status. Commands are parsed at the daemon level and either executed locally or forwarded to the container.

# Details

- Parse messages starting with "/" prefix as commands
- Lookup commands in SQLite registry (loaded from database)
- Execute built-in commands: /new, /help, /status
- Return unknown command message for unregistered commands
- Forward non-command messages to container for AI processing
- Support command arguments (e.g., /new session)
- Return formatted help text with all available commands

# File Paths

- internal/daemon/commands/handler.go
- internal/daemon/commands/registry.go
- internal/database/queries.go
- internal/daemon/daemon.go
