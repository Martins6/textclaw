# TextClaw

AI-powered messaging daemon that connects to channels (Telegram by default) and routes messages to AI agent containers. Each user/group gets an isolated Docker workspace with its own container, while a main group has sudo-like access to all workspaces.

## Features

- **Multi-channel support**: Telegram (extensible)
- **Isolated workspaces**: Each user/group gets their own Docker container
- **Persistent memory**: SQLite database with vector embeddings for semantic search
- **Heartbeats**: Periodic proactive checks that notify users of important changes
- **Context search**: Search historical conversations using semantic similarity

## Prerequisites

- **Go 1.25+** (for building from source)
- **Docker** (for running agent containers)
- **Telegram Bot Token** (get from @BotFather)

## Installation

### Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/Martins6/textclaw/main/install.sh | bash
```

### From Source

```bash
git clone https://github.com/Martins6/textclaw.git
cd textclaw
go build -o textclaw ./cmd/daemon
sudo mv textclaw /usr/local/bin/
```

## Quick Start

### 1. Initialize TextClaw

```bash
textclaw init
```

This creates `~/.textclaw/` with:

- Database (`sqlite.db`)
- Workspace directories
- Template files (Dockerfile, setup scripts, AGENTS.md)

### 2. Configure Telegram

```bash
# Set your bot token (required)
textclaw config set telegram.token "YOUR_BOT_TOKEN"

# Optional: restrict to specific users (leave empty to allow everyone)
textclaw config set telegram.allowed_users '["user1", "user2"]'
```

### 3. Start the Daemon

```bash
textclaw daemon start
```

### 4. Talk to Your Agent

Message your Telegram bot. On first contact, TextClaw automatically:

- Creates a workspace for that user
- Spawns a Docker container
- Routes messages to the agent

## CLI Commands

| Command                             | Description                          |
| ----------------------------------- | ------------------------------------ |
| `textclaw init`                     | Initialize TextClaw (~/.textclaw/)   |
| `textclaw daemon start`             | Start the background daemon          |
| `textclaw daemon stop`              | Stop the daemon                      |
| `textclaw daemon status`            | Check if daemon is running           |
| `textclaw config get [key]`         | Get a config value                   |
| `textclaw config set [key] [value]` | Set a config value                   |
| `textclaw notify "message"`         | Send a notification (from container) |

### Config Keys

| Key                      | Description                                         |
| ------------------------ | --------------------------------------------------- |
| `telegram.token`         | Telegram bot token                                  |
| `telegram.allowed_users` | List of allowed Telegram handles (empty = everyone) |
| `container.image`        | Docker image for agent containers                   |
| `workspace.base_path`    | Base path for workspaces                            |

## Configuration

Edit `~/.textclaw/setup.toml`:

```toml
[container]
image = "textclaw/agent:latest"
volumes = [
    "./workspaces/{workspace}:/home/{user}:rw",
    "~/.textclaw/opencode-config:/home/{user}/.config/opencode:ro",
]

[workspace]
base_path = "./workspaces"

[telegram]
token = "YOUR_BOT_TOKEN"
# allowed_users: Leave empty to allow everyone, or specify handles without @
allowed_users = []  # Or: ["@user1", "@user2"]
```

## ~/.textclaw/

Workspace Structure

```
├── textclaw/           # TextClaw source (if cloned)
├── Dockerfile          # Agent container definition
├── setup.toml         # Main configuration
├── setup_linux.sh    # User customizations (persists across rebuilds)
├── opencode_install.sh
├── database/
│   └── sqlite.db     # All messages & embeddings
├── workspaces/
│   ├── main/         # Main group (sudo access to all)
│   │   ├── AGENTS.md
│   │   ├── SOUL.md
│   │   ├── TOOLS.md
│   │   └── USER.md
│   └── {user}/
│       └── ...       # Auto-created per user
├── heartbeats/       # Heartbeat configs
├── cronjobs/         # Scheduled tasks
└── models/           # Embedding models
```

## Heartbeats

Enable per-workspace periodic proactive checks by adding to the workspace's `.textclaw.json`:

```json
{
  "heartbeat": {
    "enabled": true,
    "schedule": "*/30 * * * *"
  }
}
```

Create `HEARTBEATS.md` in the workspace with instructions the agent follows during heartbeats.

## Context Search

From inside a container, search historical messages:

```bash
textclaw context similar "what did I say about python"
textclaw context search "error"
textclaw context recent --limit 10
```

## Architecture

```
┌─────────────────┐
│ Channel Listeners │
│ (Telegram, WhatsApp) │
└────────┬────────┘
         │ incoming messages
         ▼
┌─────────────────┐
│ Message Router   │
└────────┬────────┘
         │ lookup sender in sqlite
         ▼
┌─────────────────┐
│ Workspace         │
│ Auto-Provisioner  │
└────────┬────────┘
         │ ensure workspace exists
         ▼
┌─────────────────┐
│ Agent Runner     │
│ (spawn container)│
└────────┬────────┘
         │ execute + stream response
         ▼
┌─────────────────┐
│ Response back to  │
│ channel           │
└─────────────────┘
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o textclaw ./cmd/daemon

# Run daemon directly
./textclaw daemon
```

## License

MIT
