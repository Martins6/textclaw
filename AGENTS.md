# TextClaw

## Overview

TextClaw is an AI-powered messaging daemon that connects to channels (Telegram, WhatsApp) and routes messages to AI agent containers. Each user/group gets an isolated Docker workspace with its own container, while a main group has sudo-like access to all workspaces.

## Core Tech

- **Language**: Go
- **Database**: SQLite with vector embeddings (llama.cpp)
- **Container Runtime**: Docker
- **AI Agent**: OpenCode (running inside containers)
- **Communication**: HTTP (daemon ↔ container), Unix socket (container ↔ daemon for notifications)

## Documentation

Detailed documentation is available in:

- `docs/DRAFT.md` - Full architecture, philosophy, and implementation details
