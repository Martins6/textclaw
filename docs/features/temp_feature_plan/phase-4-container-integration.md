# Phase 4: Container Integration

## Problem
The daemon currently receives messages and stores them but does NOT execute them in containers. Phase 4 implements:
1. **Agent Runner** - Spawns containers, sends messages via HTTP to OpenCode server
2. **Container Lifecycle** - Start/stop containers, manage volumes
3. **Unix Socket** - Enables containers to send messages back to the daemon

## Solution Overview
Build the container integration layer that:
- Creates Docker containers per workspace
- Runs OpenCode server inside containers
- Communicates via HTTP REST API
- Enables bi-directional messaging via Unix socket

## Steps

### Step 1: Add Docker SDK Dependency
- Add the Docker client library to go.mod
- Files: `go.mod`, `go.sum` (auto-generated)

### Step 2: Create Container Manager (`internal/container/`)
- Create a new package for Docker operations
- Docker client initialization
- Image pulling
- Container create/start/stop
- Volume mounting configuration
- Files: `internal/container/container.go`

### Step 3: Implement Agent Runner (`internal/daemon/runner/runner.go`)
- Execute prompts in containers and stream responses
- Spawn container if not running
- Send HTTP POST to container's OpenCode server
- Parse response with parts[] array
- Return response to daemon for channel sending
- Files: `internal/daemon/runner/runner.go`

### Step 4: Integrate Runner into Daemon
- Connect the runner to the message handling flow
- Update daemon.go to use the runner
- Handle /new command for session management
- Send responses back through the adapter
- Files: `internal/daemon/daemon.go`

### Step 5: Create Unix Socket Server (`pkg/socket/`)
- Allow containers to send messages back to daemon
- Server listens on /var/run/textclaw/textclaw.sock
- Handle notify messages from containers
- Route to correct channel adapter
- Files: `pkg/socket/server.go`, `pkg/socket/client.go`

### Step 6: Create Unix Socket Client for CLI
- Allow the textclaw CLI (mounted in containers) to send messages
- Connect to daemon socket
- Send message + workspace ID
- Support --urgent flag
- Files: `internal/cli/notify.go` (update)

## Challenges
- Container networking: Need to expose OpenCode server port from container
- Volume permissions: Main workspace needs read-only access to other workspaces
- Socket path: Must handle path in both host and container contexts
- Container readiness: Wait for OpenCode server to be ready before sending HTTP requests

## Risk Assessment
- Level: Medium
- Key Risks: Docker not available, image pull failures, container networking issues

## Fallback Plan
- Use exec instead of HTTP for simple commands if HTTP server fails
- Add health check retries for container readiness
