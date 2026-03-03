# Overview

Phase 5 adds heartbeat scheduling for periodic workspace health checks and session management with /new command support. The daemon performs cron-based heartbeat checks while agents send notifications only when there's something worth reporting.

# Details

- Add session_id column to workspaces table for tracking active sessions
- Create per-workspace JSON configuration for heartbeat settings
- Implement cron-based Heartbeat Scheduler for periodic checks per workspace
- Add /new command handler to create fresh sessions
- Update Runner to track and persist current session per workspace
- Extend socket protocol to support heartbeat notifications
- Implement "No updates" suppression to prevent notification storms
- Add workspace config loading utilities
- Integrate scheduler into daemon with graceful shutdown
- Add verbose and suppress flags to notify command

# File Paths

- internal/database/migrations/002_add_session_id.sql
- internal/database/queries.go
- internal/config/workspace.go
- internal/daemon/heartbeat/scheduler.go
- internal/daemon/daemon.go
- internal/daemon/runner/runner.go
- pkg/socket/client.go
- pkg/socket/server.go
- internal/cli/notify.go
- cmd/daemon/main.go
