# Overview

Phase 4 implements container integration that enables the daemon to spawn Docker containers per workspace, run OpenCode server inside them, and communicate via HTTP REST API and Unix sockets for bi-directional messaging.

# Details

- Add Docker SDK dependency for container operations
- Create Container Manager package for Docker operations (image pulling, container create/start/stop)
- Implement Agent Runner to execute prompts in containers via HTTP POST to OpenCode server
- Integrate Runner into daemon message handling flow
- Create Unix Socket Server to receive notifications from containers
- Create Unix Socket Client for CLI to send messages back to daemon
- Handle container networking and volume mounting configuration
- Implement container readiness checks before sending HTTP requests

# File Paths

- internal/container/container.go
- internal/daemon/runner/runner.go
- internal/daemon/daemon.go
- pkg/socket/server.go
- pkg/socket/client.go
- internal/cli/notify.go
