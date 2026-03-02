# Implementation Phases

## Phase 1: Foundation (Week 1)

1. Project Setup - Initialize Go module, create folder structure
2. Database Layer - SQLite schema, migrations, queries
3. Config Management - .textclaw/ structure, setup.toml parsing

## Phase 2: CLI Commands (Week 2)

4. Init command - Create ~/.textclaw/ structure
5. Config command - get/set config values
6. Daemon command - start/stop daemon
7. Notify command - send messages to users

## Phase 3: Daemon Core (Week 3)

8. Listener Interface - Adapter interface design
9. Telegram Adapter - Implement Telegram bot listener
10. Message Router - Route messages to workspaces
11. Workspace Provisioner - Auto-create workspaces

## Phase 4: Container Integration (Week 4)

12. Agent Runner - Spawn containers, HTTP communication
13. Container Lifecycle - Start/stop, volume mounting
14. Unix Socket - Container → daemon messaging

## Phase 5: Features (Week 5)

15. Heartbeat Feature - Periodic checks
16. Session Management - /new command handling
