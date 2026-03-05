# Overview

Designates a Telegram user as the main/admin user with full access to all workspaces. The main user gets the root ~/.textclaw directory as their workspace instead of a workspace subdirectory.

# Details

- Add [main] section to setup.toml with enabled and telegram_id fields
- Create IsMainUser() helper in config.go for case-insensitive main user detection
- Main user workspace path uses root ~/.textclaw instead of ~/.textclaw/workspaces/{workspace_id}
- Main user container mounts entire ~/.textclaw directory for access to all workspaces
- Main user gets "main" role in database instead of "user"
- Provisioner creates empty workspace ID for main user to use root directory
- Runner and container integration handle empty workspace ID gracefully
- Works alongside main group feature for group chats with elevated privileges

# File Paths

- internal/config/config.go (MainConfig struct, IsMainUser helper)
- internal/cli/config.go (config get/set for main.enabled, main.telegram_id)
- internal/daemon/provisioner/provisioner.go (main user workspace handling)
- internal/daemon/runner/runner.go (workspace path logic for main user)
- internal/daemon/daemon.go (main user initialization and directory setup)
