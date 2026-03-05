# Overview

Designates a Telegram user as the main/admin user with full access to all workspaces. The main user gets the root ~/.textclaw directory as their workspace instead of a workspace subdirectory.

# Details

- Add [main] section to setup.toml with enabled and telegram_id fields
- Create IsMainUser() helper in config.go for case-insensitive main user detection
- Main user workspace path uses root ~/.textclaw instead of ~/.textclaw/workspaces/{workspace_id}
- Main user container mounts entire ~/.textclaw directory for access to all workspaces
- Main user gets "main" role in database instead of "user"
- Provisioner creates "main" workspace ID for main user to use root directory
- Runner and container integration handle "main" workspace ID gracefully
- Fixed: Main user routing bug - workspace ID comparison now uses "main" string instead of mainUserID to correctly route to root ~/.textclaw
- Works alongside main group feature for group chats with elevated privileges

# Configuration

Add to `~/.textclaw/setup.toml`:

```toml
[main]
enabled = true
telegram_id = "your_telegram_username"
agent = "build"              # Agent to use (e.g., "build", "general")
provider = "minimax"         # AI provider (e.g., "minimax", "opencode")
model = "MiniMax-M2.5"      # Model ID from the provider
```

# File Paths

- internal/config/config.go (MainConfig struct, IsMainUser helper)
- internal/cli/config.go (config get/set for main.enabled, main.telegram_id, main.agent, main.provider, main.model)
- internal/daemon/provisioner/provisioner.go (main user workspace handling)
- internal/daemon/runner/runner.go (workspace path logic for main user, agent/model defaults)
- internal/daemon/daemon.go (main user initialization and directory setup)
