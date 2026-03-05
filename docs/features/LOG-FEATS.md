2026-03-05-00-00 | Added mount all TextClaw directories feature - mounts database, heartbeats, cronjobs, models, .opencode, textclaw, and logs directories to main user container for full workspace access
2026-03-04-17-02 | Added daemon logging feature - writes all daemon events and workspace interactions to ~/.textclaw/logs/{workspace_id}/{YYYY-MM-DD}.log with textclaw daemon logs CLI command for retrieval and live tail
2026-03-04-16-22 | Added main user/group feature - designates a Telegram user as main/admin with full access to all workspaces, uses root ~/.textclaw directory instead of workspace subdirectory, and can access all workspace containers
2026-03-04-14-07 | Added slash command system with /new, /help, /status commands and SQLite command registry
2026-03-04-14-07 | Updated README with OpenCode authentication documentation for setting up AI provider credentials
2026-03-04-10-49 | Added pre-start containers feature to daemon startup for faster message handling
2026-03-03-07-54 | Added Phase 1 Foundation feature with project setup, SQLite database layer, and TOML configuration
2026-03-03-07-54 | Added Phase 2 CLI Commands feature with config, daemon, and notify subcommands
2026-03-03-07-54 | Added Phase 3 Daemon Core feature with listener interface, Telegram adapter, router, and provisioner
2026-03-03-09-00 | Added Container Integration feature with Docker container management, Agent Runner, and Unix Socket communication
2026-03-03-09-00 | Added Heartbeat Sessions feature with cron-based scheduler and /new command for fresh sessions
2026-03-03-09-00 | Added Context Search feature with sqlite-vec embeddings and llama.cpp for historical memory search
