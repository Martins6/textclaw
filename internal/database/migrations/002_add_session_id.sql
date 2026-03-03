-- Add session_id column to workspaces table for session management

ALTER TABLE workspaces ADD COLUMN session_id TEXT;
CREATE INDEX IF NOT EXISTS idx_workspaces_session ON workspaces(session_id);
