2026-03-03-18-15 | Fixed Docker volume path error by using absolute paths instead of relative paths in config initialization
2026-03-03-18-50 | Fixed container startup timeout by adding port bindings and HTTP health check to WaitForPort function
2026-03-03-19-15 | Fixed macOS code signing issue causing textclaw init to be killed with SIGKILL on Apple Silicon - removed old binary before copying, added --deep flag to codesign, added -ldflags -s to go build
