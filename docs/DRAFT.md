# TextClaw - Philosophy & Architecture

## Philosophy

- i/o is always text.
- security by isolation.
- the actual computation is always done in a user customized dockerfile (container)
- main group will have access to all workspaces
- each group will have its own container with its own workspace
- single database for everything, it is sqlite by default with vector embeddings
- the ai agent can change everything:
  - structural changes are done via skills applied to .textclaw/textclaw folder

## I/O

Basic IO idea:

channels/groups <-> sqlite db
sqlite db <-> agent runner (container)

Channels could be telegram, whatsapp, etc.. Groups are where the agent is deployed in those channels.

There will be a main channel/group which will have like sudo like powers over the whole system. All the other will have group level access to their own workspace.

### Text vs Non-Text Handling

Every message that comes through the channel/group is classified as either **text** or **non-text**:

- **Text messages**: Stored directly in SQLite `content` column
- **Non-text (files)**: Saved to workspace filesystem, SQLite stores the file path

| content_type | content value                   | Storage    |
| ------------ | ------------------------------- | ---------- |
| `text`       | actual text content             | SQLite     |
| `non-text`   | `/workspace/files/filename.ext` | workspace/ |

The `content_type` column tells the agent if content is text or a file. Skills can then inspect the file extension to handle specific types (transcription for audio/video, OCR for images, etc).

This way any multi-modal data that can come through, the model can deal with it appropriately based on installed skills.

The metadata tagging will be done by a certain module, and you can also modify those to hide the saving of metadata or increase more metadata saving.

## Configs and Workspaces

~/.textclaw/

- textclaw/ : original git repo with Golang CLI package with all the logic of textclaw implementation, the daemon architecture
- Dockerfile: Agent Runner -> Linux VM from setup.toml with opencode_install.sh + setup_linux.sh, blocked from changes
- opencode_install.sh: blocked from changes
- setup_linux.sh: should be modified to the needs of main group
- setup.toml: linux vm type config, volume accessible to Agent Runner, folder permissions
- .opencode/
- opencode.json
- AGENTS.md
- SOUL.md
- TOOLS.md
- USER.md
- HEARTBEATS.md
- CRONJOBS.md
- workspaces/
  - group1
    - AGENTS.md
    - SOUL.md
    - TOOLS.md
    - USER.md
    - CLI installed to retrieve memories from sqlite.db but only those that belong to group1
  - group2
- heartbeats/
- cronjobs/
- database/
  - sqlite.db: database from sqllite
  - migrations/

## Container Architecture

One container per workspace, with main group container mounting ALL workspaces.

Each workspace gets its own isolated container with its own:

- Volume (workspace files)
- AGENTS.md (soul.md + tools.md + user.md + heartbeats.md + cronjobs.md)
- Database namespace (tables prefixed by workspace_id)

The main group's container additionally mounts all other workspaces as read-only volumes.

```
Main Group Container:
  volumes:
    - ~/.textclaw/workspaces/main:/home/main:rw
    - ~/.textclaw/workspaces/team-a:/home/main/workspaces/team-a:ro
    - ~/.textclaw/workspaces/team-b:/home/main/workspaces/team-b:ro

Sub-group Container (team-a):
  volumes:
    - ~/.textclaw/workspaces/team-a:/home/team-a:rw
```

This way:

- Each group has its own isolated working environment
- Main group can read/write to its own workspace, and read (but not write) to others
- Containers are completely isolated from each other (no cross-container access)
- Main group accesses other workspaces via volume mounts, not via database/API

## Daemon Architecture

A single Go daemon process (`textclaw-daemon`) that runs continuously, listening to all channels.

```
┌─────────────────┐
│  Channel Listeners   │
│  (Telegram, WhatsApp) │
└────────┬────────┘
         │ incoming messages
         ▼
┌─────────────────┐
│  Message Router      │
│  (route to workspace)│
└────────┬────────┘
         │ lookup sender in sqlite
         ▼
┌─────────────────┐
│  Workspace            │
│  Auto-Provisioner     │
│  (on new sender)      │
└────────┬────────┘
         │ ensure workspace exists
         ▼
┌─────────────────┐
│  Agent Runner        │
│  (spawn container)   │
└────────┬────────┘
         │ execute + stream response
         ▼
┌─────────────────┐
│  Response back to      │
│  channel               │
└─────────────────┘
```

### Components

**Channel Listeners**

- Telegram adapter
- WhatsApp adapter
- (extensible interface for more)
- Each listens for messages, normalizes to internal format

**Message Router**

- Looks up sender/chat_id in sqlite `contacts` table
- Routes message to correct workspace based on contact -> workspace mapping

**Workspace Auto-Provisioner**

- If sender unknown → creates everything:
  - `workspaces/{sender_handle}/` directory
  - Copies AGENTS.md from template
  - Registers in sqlite `workspaces` table
  - Creates container config
- Triggered on first message from new contact

**Agent Runner**

- Manages container lifecycle (start, stop, stream)
- Communicates with container via HTTP (opencode server API)
- Sends prompt to container's opencode server: `POST /session/:id/message`
- Receives response as JSON with `parts[]` array
- Streams response back to channel

**HTTP Communication Flow:**

```
Daemon                           Container (opencode serve)
   │                                      │
   │  POST /session/:id/message           │
   │  { "parts": [{ "type": "text",       │
   │     "text": "Hello" }] }             │
   ├─────────────────────────────────────►│
   │                                      │ (processes prompt)
   │  { "info": {...}, "parts": [...] }  │
   │◄─────────────────────────────────────┤
   │                                      │
```

**Key opencode server endpoints used:**

| Endpoint | Purpose |
|----------|---------|
| `POST /session` | Create new session (via `/new` command) |
| `POST /session/:id/message` | Send prompt, get response |
| `GET /session/:id/message` | Get conversation history |
| `GET /session` | List sessions |

SQLite is the **source of truth** - all conversation history persists there. The container's session is ephemeral working state.

**SQLite Database Schema (key tables)**

```sql
CREATE TABLE contacts (
    id TEXT PRIMARY KEY,        -- telegram handle, whatsapp number, etc.
    workspace_id TEXT,
    role TEXT,                  -- 'main', 'admin', 'user'
    created_at TIMESTAMP
);

CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,        -- slug: "main", "adriel-wife", etc.
    container_id TEXT,
    created_at TIMESTAMP
);

CREATE TABLE messages (
    id INTEGER PRIMARY KEY,
    workspace_id TEXT,
    contact_id TEXT,
    content TEXT,               -- text content or file path
    content_type TEXT,          -- 'text' or 'non-text'
    direction TEXT,             -- 'incoming' or 'outgoing'
    timestamp TIMESTAMP
);
```

### Flow

1. Telegram message arrives: "Hello from @wife"
2. Router checks: does @wife exist in contacts?
   - NO → Auto-provisioner creates workspace "wife", creates contact entry
   - YES → Get workspace_id
3. Router sends message to Agent Runner for workspace "wife"
4. Agent Runner spawns container, runs opencode with prompt
5. Response streams back to Telegram

This way, the daemon just needs to run. New users/groups automatically get their own workspace on first message.

### Session Management via Channel

The daemon maintains **one persistent session per workspace**. By default, all messages continue in the same session (conversation context is preserved). To create a **new session**, users must explicitly invoke a command via their channel:

| Channel | Command | Description |
| ------- | ------- |-------------|
| Telegram | `/new` or `/new session` | Start fresh session |
| WhatsApp | `/new` | Start fresh session |
| CLI | `textclaw session new` | Start fresh session |

**How it works:**

1. User sends `/new` via their channel
2. Daemon calls `POST /session` to create new session in container
3. New session ID stored in SQLite for that workspace
4. Subsequent messages use the new session (context cleared)

**If no explicit `/new`:**
- All messages continue in the existing session (default behavior)
- Conversation history is preserved across messages

This design:
- Keeps sessions tied to channel capabilities (Telegram/WhatsApp have different command syntax)
- Lets users control when they want fresh context
- Prevents accidental context bleeds between topics

## Container Lifecycle & Persistence

The agent container runs as long as the daemon is running. When the daemon stops, the container is destroyed and recreated on next start.

### Key behaviors

- **Container stays up while daemon runs** - All changes persist (CLI installs, file edits, etc.)
- **Container rebuilds on daemon restart/crash** - All changes are lost
- **Only workspace volume persists** - Files in `~/.textclaw/workspaces/{group}/` survive

### Agent instructions

Add to AGENTS.md (or group-specific AGENTS.md) as a reminder:

> "If you install anything system-level (CLI tools, packages, etc.), add it to setup_linux.sh so it survives container rebuilds. Temporary changes for testing are fine, but final changes must be baked into the container build."

This way the agent knows to:

1. Install and test in the running container
2. If it works, modify setup_linux.sh to persist the change
3. Warn the user if system-level changes aren't baked in

## Onboarding

### Prerequisites

1. **OpenCode** - User already has OpenCode set up on host via `opencode auth login`
2. **Telegram Bot Token** - Get from @BotFather on Telegram

### Setup Flow

```bash
# 1. Initialize TextClaw (creates ~/.textclaw/ structure)
textclaw init

# 2. Configure Telegram bot
textclaw config set telegram.token "123:ABC..."

# 3. Configure allowed users (optional - defaults to all)
textclaw config set telegram.allowed_users "@wife,@friend1,@friend2"

# 4. Start daemon
textclaw daemon start
```

### OpenCode Credentials in Container

OpenCode runs **inside** the container, not on the host. The daemon mounts the host's OpenCode auth into each workspace container.

```toml
# setup.toml
[container]
volumes = [
    "./workspaces/{workspace}:/home/{user}:rw",
    # Mount textclaw's isolated OpenCode config (read-only)
    "~/.textclaw/opencode-config:/home/{user}/.config/opencode:ro",
]
```

The container automatically has access to OpenCode because it sees the same auth path the host uses.

**Telegram token:**

- Telegram bot token stays on **host** (daemon listens to Telegram)
- Daemon passes message content to container
- Container doesn't need Telegram token - it just receives prompts

## Heartbeat Feature

Heartbeats are periodic, scheduled checks that transform TextClaw from a reactive chatbot into a proactive always-on assistant. Unlike OpenClaw's minimal "HEARTBEAT_OK" response, TextClaw's heartbeats are more verbal and contextual.

### How It Works

1. **Activation**: User enables heartbeats via CLI: `textclaw heartbeat enable --every 30m`
2. **Instruction Reading**: Agent reads HEARTBEAT.md for what checks to perform
3. **Lightweight Checks**: Agent runs deterministic checks (file existence, simple queries, etc.)
4. **Context Awareness**: Agent also checks recent conversation for user hints about things they want monitored
5. **Smart Reporting**: Only sends a message to user if there's something worth reporting

### HEARTBEAT.md Format

```markdown
# Heartbeat Instructions

## Always Check

- Check if any files need attention in /workspace/pending/
- Check cronjob status from cronjobs/ directory

## Conditional Checks (if mentioned by user)

- Check {specific topic} if user mentioned wanting updates on it
- Monitor {specific process} if user asked to track it
```

### Sending Messages to User

The agent uses the TextClaw CLI to send heartbeat results:

```bash
# Send message to the user (via the channel they're on)
textclaw notify "Your summary of findings..."

# Or with more detail
textclaw notify --urgent "Disk space is low: 5% free"
```

### Activation

Heartbeats are configured in `.textclaw.json`, not via CLI. Just set `heartbeat.enabled: true` and `heartbeat.every` with your desired interval.

### Heartbeat Response Behavior

- **No issues found**: Agent stays silent (no "HEARTBEAT_OK" spam)
- **Something to report**: Agent sends a concise notification with findings
- **User hints**: If user said "let me know if X changes", agent checks for X changes

This approach keeps heartbeats lightweight but still proactive - the agent does the work silently and only speaks up when there's actual value to share.

## High-Level Architecture

```
textclaw/
├── cmd/
│   ├── textclaw/           # Main CLI entrypoint
│   │   └── main.go
│   ├── daemon/             # textclaw daemon
│   │   └── main.go
│   └── serve/              # Internal API server (for container communication)
├── internal/
│   ├── cli/                # CLI commands (init, config, notify, session)
│   ├── daemon/             # Core daemon logic
│   │   ├── listener/       # Channel adapters (telegram, whatsapp)
│   │   ├── router/         # Message routing to workspaces
│   │   ├── provisioner/    # Auto-provision workspaces
│   │   └── runner/         # Container lifecycle management
│   ├── container/          # Docker/container operations
│   ├── database/           # SQLite operations + migrations
│   ├── config/             # Config loading/writing
│   └── workspace/          # Workspace management
├── pkg/
│   ├── textclaw/           # Shared Go module (can be imported)
│   └── socket/             # Unix socket client/server for daemon<->container comm
└── scripts/                # Setup scripts, build scripts
```

## Key Modules

| Module | Responsibility |
|--------|---------------|
| `cmd/textclaw` | CLI entrypoint (init, config, daemon, notify) |
| `cmd/daemon` | Long-running process listening to Telegram/WhatsApp |
| `internal/cli` | Implements each CLI subcommand |
| `internal/daemon/listener` | Telegram adapter, WhatsApp adapter (interface-based) |
| `internal/daemon/router` | Routes messages to workspace based on contact |
| `internal/daemon/provisioner` | Creates workspaces on first message |
| `internal/daemon/runner` | Spawns containers, communicates via HTTP |
| `internal/database` | SQLite schema, migrations, queries |
| `pkg/socket` | Unix socket for container → daemon messaging |

## Daemon Flow (detailed)

```
incoming message → listener.Normalize() → router.Lookup() → 
  (if new: provisioner.Create()) → runner.Execute() → response → listener.Send()
```

## Package Structure

```
internal/
├── cli/
│   ├── init.go         # textclaw init
│   ├── config.go      # textclaw config set/get
│   ├── notify.go      # textclaw notify
│   ├── session.go     # textclaw session new/list
│   └── daemon.go      # textclaw daemon start/stop
├── daemon/
│   ├── daemon.go      # Main daemon loop
│   ├── listener/
│   │   ├── adapter.go # interface Adapter { Listen(), Send() }
│   │   ├── telegram.go
│   │   └── whatsapp.go
│   ├── router.go
│   ├── provisioner.go
│   └── runner.go
├── database/
│   ├── db.go          # *sql.DB connection
│   ├── migrations/    # SQL migration files
│   └── queries.go     # Prepared statements
└── workspace/
    └── workspace.go   # Workspace CRUD operations
```

## Communication Patterns

1. **Daemon → Container**: HTTP REST to `localhost:PORT/session/:id/message`
2. **Container → Daemon**: Unix socket (`/var/run/textclaw/textclaw.sock`)
3. **CLI → Daemon**: Same Unix socket for `notify`, `session`, `context` commands

## Context Search (Historical Memory)

Each workspace container needs to search conversation history for context - finding previous messages that are semantically similar, keyword matches, or recent conversations. This is achieved through the existing Unix socket infrastructure.

### Why Not Direct SQLite Access?

- Mounting SQLite directly to containers would bypass workspace isolation
- A container could query other workspace's data
- Socket-based approach enforces workspace_id filtering at the daemon level

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│ Container (workspace: team-a)                               │
│                                                             │
│   textclaw context similar "what did I say about python"   │
│          │                                                  │
│          ▼                                                  │
│   CLI reads ~/.textclaw.json → gets workspace_id: team-a  │
│          │                                                  │
│          ▼                                                  │
│   Socket message: "CONTEXT_SEARCH|semantic|what did I..."   │
│          │                                                  │
│          ▼ (Unix socket /var/run/textclaw/textclaw.sock)  │
└────────────┼────────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────────┐
│ Daemon                                                     │
│   1. Parse socket message                                  │
│   2. Extract workspace_id from request                     │
│   3. If semantic: generate embedding for query            │
│   4. Query SQLite (with workspace_id filter)              │
│   5. Return results                                         │
└─────────────────────────────────────────────────────────────┘
```

### Socket Message Format

```
CONTEXT_SEARCH|<search_type>|<query>

Where:
- search_type: semantic, keyword, recent
- query: the actual search query
```

**Examples:**

```
CONTEXT_SEARCH|semantic|what did I say about debugging
CONTEXT_SEARCH|keyword|python
CONTEXT_SEARCH|recent|10
```

### CLI Commands

The container mounts a `textclaw` CLI binary that provides context search:

```bash
# Semantic search - find messages similar to query
textclaw context similar "what did I say about debugging errors"

# Keyword search - find messages containing term
textclaw context search "python"

# Recent messages - get last N messages
textclaw context recent --limit 10

# Full-text search
textclaw context find "error handling"
```

### Database Schema

Using `sqlite-vec` extension for vector similarity search:

```sql
-- Messages table (existing)
CREATE TABLE messages (
    id INTEGER PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    contact_id TEXT,
    content TEXT,
    content_type TEXT,
    direction TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Vector embeddings using sqlite-vec
CREATE VIRTUAL TABLE message_embeddings USING vec0(
    embedding float[384]  -- dimension depends on embedding model
);

-- Indexes
CREATE INDEX idx_messages_workspace ON messages(workspace_id);
CREATE INDEX idx_embeddings_workspace ON message_embeddings(workspace_id);
```

### Semantic Search Query (sqlite-vec)

```sql
SELECT 
    m.content,
    m.timestamp,
    v.distance as similarity
FROM message_embeddings v
JOIN messages m ON m.id = v.message_id
WHERE v.workspace_id = ?
ORDER BY v.distance
LIMIT 10;
```

### Security & Isolation

- **No direct DB access**: Containers cannot mount SQLite read-write
- **Enforced filtering**: All queries auto-filtered by workspace_id from `~/.textclaw.json`
- **Read-only**: Only SELECT queries allowed (no INSERT/UPDATE/DELETE via socket)
- **Main group sudo**: Main workspace can optionally query all workspaces with special flag

### Embedding Generation

- Uses local embedding model (e.g., sentence-transformers via Go binding or subprocess)
- Generated asynchronously when messages are stored
- Query embedding generated on-demand during search

### This Complements Sessions

- **Sessions**: Short-term working memory (in container, ephemeral)
- **Context Search**: Long-term historical memory (in SQLite, persistent)

This gives agents powerful context retrieval: *"Oh, you mentioned wanting to track X yesterday..."*

---

The container needs a way to send messages back to the user. This is solved by mounting the TextClaw CLI and a per-workspace config into each container.

### Setup

The daemon mounts these into each workspace container:

```toml
# setup.toml
[container]
volumes = [
    "./workspaces/{workspace}:/home/{user}:rw",
    # TextClaw CLI + config for outbound messages
    "./textclaw:/usr/local/bin/textclaw:ro",
    "./workspaces/{workspace}/textclaw.json:/home/{user}/.textclaw.json:ro",
]
```

### Per-Workspace Config

Each workspace has a `~/.textclaw.json` config that includes heartbeat settings:

```json
{
    "workspace": "wife",
    "channel": "telegram",
    "target": "@wife_handle",
    "heartbeat": {
        "enabled": true,
        "every": "30m"
    }
}
```

- `channel`: "telegram", "whatsapp", etc.
- `target`: the user identifier (phone number, Telegram handle)
- `heartbeat.enabled`: true/false to enable periodic checks
- `heartbeat.every`: duration string ("30m", "1h", "10m", etc.)

### How `textclaw notify` Works

1. Agent runs `textclaw notify "message"` in container
2. CLI reads `~/.textclaw.json` to know destination
3. CLI connects to daemon via Unix socket (mounted from host)
4. Daemon receives message and forwards to Telegram/WhatsApp
5. User gets the notification in their chat

### Why This Works

- **No separate API needed**: Uses Unix socket mounted into container
- **Per-workspace config**: Each workspace knows its target user/channel
- **CLI is lightweight**: Just a Go binary, minimal overhead
- **Agent stays isolated**: Can't bypass the socket - only sends through daemon
