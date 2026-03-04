# Overview

TextClaw is an AI-powered messaging daemon that connects to messaging channels (Telegram, WhatsApp) and routes messages to AI agent containers. Each user/group gets an isolated Docker workspace with its own container, while a main group has sudo-like access to all workspaces.

# Files

- feat-foundation.md - Project setup, database layer with SQLite, and configuration management with TOML
- feat-cli-commands.md - CLI commands for config, daemon lifecycle, and notify functionality
- feat-daemon-core.md - Listener interface, Telegram adapter, message router, and workspace provisioner
- feat-container-integration.md - Docker container management, Agent Runner, and Unix Socket communication
- feat-heartbeat-sessions.md - Cron-based heartbeat scheduler and session management with /new command
- feat-context-search.md - Vector search with sqlite-vec embeddings and llama.cpp for historical memory
- feat-slash-commands.md - Slash command system with /new, /help, /status and SQLite command registry
