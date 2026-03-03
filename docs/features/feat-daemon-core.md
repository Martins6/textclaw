# Overview

Phase 3 implements the core daemon functionality including listener adapter interface, Telegram bot integration, message routing to workspaces, and automatic workspace provisioning.

# Details

- Define Adapter interface with Listen, Send, and Name methods
- Define normalized Message struct with sender, content, timestamp, chatID
- Define MessageHandler interface for handling incoming messages
- Implement TelegramAdapter using Telegram Bot API with long polling
- Implement Send method to send messages back to users
- Handle file attachments (photos, documents) - save to workspace, store path in DB
- Create Router struct to look up sender in SQLite contacts table
- Implement Router.Lookup to return workspace_id
- Create Provisioner struct with database connection and workspace base path
- Implement EnsureWorkspace to create workspace directory, copy templates, insert into DB
- Integrate all components into daemon main with graceful shutdown
- Add Telegram Bot API dependency and structured logging

# File Paths

- internal/daemon/listener/adapter.go
- internal/daemon/listener/telegram.go
- internal/daemon/router/router.go
- internal/daemon/provisioner/provisioner.go
- internal/daemon/daemon.go
- cmd/daemon/main.go
