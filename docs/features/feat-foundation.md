# Overview

Phase 1 establishes the core infrastructure of TextClaw - the Go project structure, SQLite database layer with migrations, and configuration management from TOML files.

# Details

- Initialize Go module with proper folder structure per architecture spec
- Create cmd/textclaw/main.go for CLI entrypoint
- Create cmd/daemon/main.go for daemon entrypoint
- Install dependencies: go-sqlite3, BurntSushi/toml, spf13/cobra
- Define SQLite schema with contacts, workspaces, and messages tables
- Create database connection management and CRUD operations
- Define Config struct with Container and Workspace configuration
- Create setup.toml parsing for configuration
- Create .textclaw/ directory structure template with init command
- Verify project builds successfully

# File Paths

- go.mod
- cmd/textclaw/main.go
- cmd/daemon/main.go
- internal/cli/init.go
- internal/database/db.go
- internal/database/migrations/001_initial.sql
- internal/database/queries.go
- internal/config/config.go
- internal/config/setup.go
- internal/workspace/
- pkg/
- templates/
