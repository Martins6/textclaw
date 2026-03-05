2026-03-04-21-50 | Fixed JSON decoding error by correcting OpenCode API field names - use "text" for request and response, added HTTP client with no timeout, added empty response handling, added retry logic with exponential backoff, and added session auto-recreation on invalid/empty responses
2026-03-04-14-07 | Fixed JSON array config escaping by adding JSON parsing helper for telegram.allowed_users and container.volumes
2026-03-04-14-07 | Fixed textclaw init missing .opencode directory by adding the directory to the dirs slice in init.go
2026-03-04-11-00 | Fixed "invalid session" error by correcting session ID parsing to use root-level `id` field instead of `info.session_id`
2026-03-04-11-00 | Fixed container health check using wrong port by getting Docker-assigned host port immediately after container starts and using it for health checks
2026-03-04-10-49 | Fixed workspace SQLite isolation by using workspace-specific state directories instead of shared ~/.textclaw/opencode-state/
2026-03-04-10-49 | Fixed 500 internal server error on session creation by implementing dynamic port allocation for each workspace container
2026-03-04-10-49 | Fixed container port bindings lost issue by removing and recreating stopped containers to ensure correct port bindings
2026-03-04-10-49 | Fixed health check timeout by reverting WaitForPort to use localhost and increasing timeout from 60s to 120s
2026-03-04-10-49 | Fixed Docker endpoint connection error on macOS by explicitly specifying Docker socket path for Docker Desktop
2026-03-04-10-49 | Fixed wait for port timeout by using container's internal IP for health check reliability
2026-03-04-10-49 | Fixed connection refused error by adding container state check and always waiting for port before creating session
2026-03-03-18-15 | Fixed Docker volume path error by using absolute paths instead of relative paths in config initialization
2026-03-03-18-50 | Fixed container startup timeout by adding port bindings and HTTP health check to WaitForPort function
2026-03-03-19-15 | Fixed macOS code signing issue causing textclaw init to be killed with SIGKILL on Apple Silicon - removed old binary before copying, added --deep flag to codesign, added -ldflags -s to go build
