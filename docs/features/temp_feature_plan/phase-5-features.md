# Phase 5: Features Implementation Plan

## Problem Statement

Phase 5 adds two key features to TextClaw:

1. **Heartbeat Feature**: Periodic health checks for each workspace using a configurable scheduler. The agent should read `HEARTBEATS.md` for instructions, perform checks, and send notifications only when there's something worth reporting via the `textclaw notify` CLI command.

2. **Session Management**: Add support for the `/new` command to create fresh sessions while preserving the default behavior of continuing existing sessions. The current session ID needs to be stored in the database.

## Solution Overview

The implementation adds:

- **Database field**: `session_id` column in the `workspaces` table to track active sessions
- **Per-workspace config**: JSON configuration file (`workspaces/{id}/.textclaw.json`) for heartbeat settings
- **Heartbeat scheduler**: A cron-based scheduler in the daemon that triggers periodic checks per workspace
- **Agent integration**: Agent reads `HEARTBEATS.md` and executes checks, sending notifications via Unix socket
- **Command handler**: `/new` command creates fresh sessions via `POST /session`

---

## Steps to Solve

### Step 1: Database Migration - Add session_id to workspaces

**Description:** Create a new migration to add `session_id` column to the workspaces table for tracking active sessions.

**Files to Change:**
- `internal/database/migrations/002_add_session_id.sql` (new file)
- `internal/database/queries.go` (add UpdateWorkspaceSession, GetWorkspaceSession functions)

**Dependencies:** None (first step)

**Estimated Effort:** Small

**Details:**
```sql
-- internal/database/migrations/002_add_session_id.sql
ALTER TABLE workspaces ADD COLUMN session_id TEXT;
CREATE INDEX idx_workspaces_session ON workspaces(session_id);
```

Add query functions:
- `UpdateWorkspaceSession(db *DB, workspaceID, sessionID string) error`
- `GetWorkspaceSession(db *DB, workspaceID string) (string, error)`

---

### Step 2: Per-Workspace Configuration Structure

**Description:** Define the JSON structure for per-workspace heartbeat configuration and create utilities to load it.

**Files to Change:**
- `internal/config/workspace.go` (new file)
- Update `internal/daemon/daemon.go` to use workspace config

**Dependencies:** Step 1

**Estimated Effort:** Small

**Details:**

Config file location: `workspaces/{workspace_id}/.textclaw.json`

```json
{
  "heartbeat": {
    "enabled": true,
    "schedule": "*/5 * * * *",
    "notify_on": ["changes", "errors", "summary"]
  },
  "agent": {
    "read_heartbeats": true
  }
}
```

Create `internal/config/workspace.go`:
```go
type WorkspaceConfig struct {
    Heartbeat *HeartbeatConfig `json:"heartbeat,omitempty"`
    Agent     *AgentConfig     `json:"agent,omitempty"`
}

type HeartbeatConfig struct {
    Enabled   bool     `json:"enabled"`
    Schedule  string   `json:"schedule"`  // cron expression
    NotifyOn  []string `json:"notify_on"` // changes, errors, summary
}

type AgentConfig struct {
    ReadHeartbeats bool `json:"read_heartbeats"`
}

func LoadWorkspaceConfig(workspacePath string) (*WorkspaceConfig, error)
func SaveWorkspaceConfig(workspacePath string, cfg *WorkspaceConfig) error
```

---

### Step 3: Create Heartbeat Scheduler

**Description:** Implement a cron-based scheduler that manages heartbeat jobs for each workspace with heartbeat enabled.

**Files to Change:**
- `internal/daemon/heartbeat/scheduler.go` (new file)
- `internal/daemon/daemon.go` (integrate scheduler)

**Dependencies:** Step 2, robfig/cron library

**Estimated Effort:** Medium

**Details:**

Create `internal/daemon/heartbeat/scheduler.go`:
```go
package heartbeat

type Scheduler struct {
    cron     *cron.Cron
    jobs     map[string]cron.EntryID  // workspaceID -> jobID
    runner   *runner.Runner
    db       *database.DB
    cfg      *config.Config
}

func NewScheduler(runner *runner.Runner, db *database.DB, cfg *config.Config) *Scheduler
func (s *Scheduler) Start(ctx context.Context) error
func (s *Scheduler) Stop() error
func (s *Scheduler) AddWorkspace(workspaceID, schedule string) error
func (s *Scheduler) RemoveWorkspace(workspaceID string) error
func (s *Scheduler) TriggerHeartbeat(ctx context.Context, workspaceID string) error
```

**Scheduler behavior:**
1. On startup, load all workspaces with heartbeat enabled from DB
2. Add cron jobs for each enabled workspace
3. On heartbeat trigger: call runner.Execute() with heartbeat prompt
4. Parse agent response; if worth reporting, call notify

---

### Step 4: Implement /new Command Handler

**Description:** Add handling for `/new` command in the daemon message flow to create fresh sessions.

**Files to Change:**
- `internal/daemon/runner/runner.go` (add NewSession method)
- `internal/daemon/daemon.go` (add /new command detection)

**Dependencies:** Step 1

**Estimated Effort:** Small

**Details:**

In `internal/daemon/daemon.go`, modify `handleMessage`:
```go
func (d *Daemon) handleMessage(ctx context.Context, msg *listener.Message) error {
    // ... existing code ...
    
    // Check for /new command
    if strings.HasPrefix(msg.Content, "/new") {
        sessionID, err := d.runner.NewSession(ctx, workspaceID)
        if err != nil {
            return d.adapter.Send(msg.From, "Failed to create new session: "+err.Error())
        }
        // Update session_id in database
        database.UpdateWorkspaceSession(d.db, workspaceID, sessionID)
        return d.adapter.Send(msg.From, "Started a new session. Previous context cleared.")
    }
    
    // ... rest of existing code ...
}
```

In `internal/daemon/runner/runner.go`, enhance:
```go
func (r *Runner) NewSession(ctx context.Context, workspaceID string) (string, error) {
    // Ensure container is running
    ip, err := r.ensureContainer(ctx, workspaceID)
    if err != nil {
        return "", err
    }
    
    // Create new session via HTTP
    resp, err := r.httpClient.PostForm(
        fmt.Sprintf("http://%s:8080/session", ip),
        url.Values{},
    )
    // Parse response to get session_id
    // Return sessionID
}
```

---

### Step 5: Update Runner to Track Sessions

**Description:** Modify the runner to track the current session per workspace and persist it to the database.

**Files to Change:**
- `internal/daemon/runner/runner.go` (track current session)
- `internal/database/queries.go` (add session persistence)

**Dependencies:** Step 1

**Estimated Effort:** Small

**Details:**

Add to runner:
```go
type Runner struct {
    // ... existing fields ...
    currentSessions map[string]string  // workspaceID -> sessionID
    mu             sync.RWMutex
}

func (r *Runner) GetCurrentSession(workspaceID string) string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.currentSessions[workspaceID]
}

func (r *Runner) SetCurrentSession(workspaceID, sessionID string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.currentSessions[workspaceID] = sessionID
}
```

---

### Step 6: Agent Heartbeat Integration

**Description:** Update the agent template to include heartbeat instructions and ensure the agent can trigger heartbeat checks.

**Files to Change:**
- `templates/.opencode/HEARTBEATS.md` (enhance instructions)
- Update provisioning to copy HEARTBEATS.md to workspace

**Dependencies:** Step 2

**Estimated Effort:** Small

**Details:**

Enhance `templates/.opencode/HEARTBEATS.md`:
```markdown
# Heartbeats

Periodic checks to perform when heartbeat is enabled.

## How Heartbeats Work

1. You will receive a heartbeat trigger with the current state
2. Read this HEARTBEATS.md file for what to check
3. Perform your checks and determine if there's anything worth reporting
4. Use `textclaw notify` only if there's something meaningful

## Always Check

- Check if any files need attention in /workspace/pending/
- Check cronjob status from cronjobs/ directory
- Review recent activity since last heartbeat

## Conditional Checks

- Check {specific topic} if user mentioned wanting updates on it

## Reporting Format

If there's something worth reporting, respond with:
- What changed
- Why it matters
- Any action needed

If nothing worth reporting, respond with "No updates" (this will NOT trigger a notification).
```

---

### Step 7: CLI Notify Enhancement

**Description:** Extend the notify command to support workspace-specific configuration and better error handling.

**Files to Change:**
- `internal/cli/notify.go` (enhance for heartbeat responses)
- Add support for reading workspace-specific config

**Dependencies:** Step 2

**Estimated Effort:** Small

**Details:**

Enhance notify.go to:
1. Accept workspace ID as argument (optional, defaults to config)
2. Support heartbeat response mode (suppress if "No updates")
3. Add verbose mode for debugging

```go
cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace ID")
cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
cmd.Flags().BoolVarP(&suppress, "suppress", "s", false, "Suppress if no updates")
```

---

### Step 8: Socket Protocol Enhancement

**Description:** Extend the Unix socket protocol to support heartbeat notifications from containers.

**Files to Change:**
- `pkg/socket/client.go` (add HeartbeatNotify method)
- `pkg/socket/server.go` (handle heartbeat notifications)

**Dependencies:** Step 7

**Estimated Effort:** Small

**Details:**

Add to socket protocol:
```go
// Client method
func (c *Client) SendHeartbeatNotify(workspaceID, message, target string, suppressNoUpdates bool) error

// Server handler
case "heartbeat_notify":
    // Handle heartbeat-specific logic (check for "No updates")
```

---

### Step 9: Daemon Integration

**Description:** Wire everything together in the daemon - integrate scheduler, handle workspace config changes, graceful shutdown.

**Files to Change:**
- `internal/daemon/daemon.go` (integrate all components)
- `cmd/daemon/main.go` (ensure proper startup order)

**Dependencies:** Steps 3, 4, 5, 6, 7, 8

**Estimated Effort:** Medium

**Details:**

Modify daemon struct:
```go
type Daemon struct {
    // ... existing fields ...
    heartbeatScheduler *heartbeat.Scheduler
}
```

In `Run()`:
1. Start heartbeat scheduler after runner is ready
2. Load all workspaces and register heartbeat jobs
3. Handle workspace provisioning to add heartbeat config
4. Graceful shutdown: stop scheduler before other components

---

### Step 10: Testing and Documentation

**Description:** Create tests and document the new features.

**Files to Change:**
- `internal/daemon/heartbeat/scheduler_test.go` (new file)
- `internal/config/workspace_test.go` (new file)
- Update `docs/PHASES.md` with implementation notes

**Dependencies:** All previous steps

**Estimated Effort:** Small

**Details:**

Write unit tests for:
- Scheduler job registration and execution
- Workspace config loading/saving
- /new command handling

---

## Challenges

### Challenge 1: Cron Expression Validation
**Mitigation:** Use robfig/cron's built-in validation. Reject configs with invalid expressions at load time.

### Challenge 2: Heartbeat Race Conditions
**Mitigation:** Use mutex locks in scheduler for job management. Ensure heartbeat triggers don't overlap for same workspace.

### Challenge 3: Container Communication During Heartbeat
**Mitigation:** Add timeout to heartbeat requests (30s default). Log failures but don't block scheduler.

### Challenge 4: Session Persistence Across Daemon Restarts
**Mitigation:** Store session_id in database. On daemon start, check if existing session is still valid via container health check.

### Challenge 5: Notification Storms
**Mitigation:** Implement rate limiting per workspace (max 1 notification per minute). Use "suppress no updates" flag.

### Challenge 6: Workspace Config File Creation
**Mitigation:** Auto-create default config on first workspace provision. Provide CLI command to manage config.

---

## Risk Assessment

### Level: Medium

**Rationale:** This feature involves:
- Database schema changes (low risk with migrations)
- New background process (scheduler)
- Inter-container communication
- User-facing command behavior change

**Key Risks:**
1. **Session ID not persisted**: If container restarts, session may be lost
   - Risk: User loses conversation context unexpectedly
   - Mitigation: Store session in DB, validate on each message

2. **Heartbeat causes resource exhaustion**: Too many concurrent heartbeat checks
   - Risk: Daemon becomes unresponsive
   - Mitigation: Limit concurrent heartbeat jobs (max 10)

3. **Agent misinterprets heartbeat**: Sends unnecessary notifications
   - Risk: User gets spammed
   - Mitigation: "No updates" suppression, rate limiting

4. **Invalid cron expressions**: User provides bad config
   - Risk: Scheduler fails to start
   - Mitigation: Validate on config load, log errors clearly

---

## Fallback Plan

### Primary Approach: Use robfig/cron for scheduling

If issues arise with robfig/cron:
- **Alternative 1**: Use Go's `time.Ticker` with workspace-specific intervals
- **Alternative 2**: External cron job that calls daemon HTTP endpoint

### Session Management Fallback

If session tracking becomes problematic:
- **Alternative**: Use session ID only in memory (current behavior)
- Track but don't persist; on daemon restart, create new session

### Notification Fallback

If Unix socket notifications fail:
- **Alternative**: Use HTTP callback to daemon's HTTP endpoint
- Fall back to direct adapter Send() if socket unavailable

---

## Verification

### Test Case 1: Database Migration
```bash
# Verify migration runs without error
go run cmd/daemon/main.go
# Check that session_id column exists
sqlite3 ~/.textclaw/textclaw.db "PRAGMA table_info(workspaces);"
```

### Test Case 2: Workspace Config
```bash
# Create workspace config manually
echo '{"heartbeat":{"enabled":true,"schedule":"*/1 * * * *"}}' > workspaces/test/.textclaw.json
# Verify it loads correctly
go run cmd/textclaw/main.go config get heartbeat.enabled --workspace test
```

### Test Case 3: /new Command
```bash
# Send /new command via Telegram
/new
# Verify new session created (check logs)
# Verify response: "Started a new session. Previous context cleared."
```

### Test Case 4: Heartbeat Trigger
```bash
# Enable heartbeat with 1-minute schedule
# Wait for heartbeat to trigger
# Check daemon logs for heartbeat execution
# Verify notification sent (if worth reporting)
```

### Test Case 5: "No Updates" Suppression
```bash
# Agent responds with "No updates"
# Verify NO notification is sent to user
# Verify heartbeat scheduler continues running
```

### Manual Verification Steps
1. Start daemon with heartbeat enabled workspace
2. Wait for heartbeat interval
3. Check container logs for heartbeat execution
4. Check user received notification (or not, if "No updates")
5. Send `/new` command, verify fresh session
6. Restart daemon, verify session restored from DB

### Expected Outcomes
- Session ID persists across daemon restarts
- Heartbeat triggers at configured intervals
- Notifications sent only when agent reports something
- /new command creates fresh session with confirmation message
- All existing functionality continues to work
