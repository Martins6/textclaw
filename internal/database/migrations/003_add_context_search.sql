-- Context Search: Add sqlite-vec and FTS5 for vector and keyword search

-- Load sqlite-vec extension
SELECT load_extension('vec0');

-- Create message_embeddings virtual table for vector similarity search
-- Using 768 dimensions to match nomic-embed-text-v1.5
CREATE VIRTUAL TABLE IF NOT EXISTS message_embeddings USING vec0(
    message_id INTEGER NOT NULL,
    workspace_id TEXT NOT NULL,
    embedding float[768]
);

-- Create FTS5 virtual table for keyword/full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    workspace_id,
    content='messages',
    content_rowid='id'
);

-- Create triggers to keep FTS in sync with messages table
CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content, workspace_id) 
    VALUES (new.id, new.content, new.workspace_id);
END;

CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
    DELETE FROM messages_fts WHERE rowid = old.id;
END;

CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
    DELETE FROM messages_fts WHERE rowid = old.id;
    INSERT INTO messages_fts(rowid, content, workspace_id) 
    VALUES (new.id, new.content, new.workspace_id);
END;

-- Indexes for workspace isolation
CREATE INDEX IF NOT EXISTS idx_message_embeddings_workspace ON message_embeddings(workspace_id);
CREATE INDEX IF NOT EXISTS idx_messages_fts_workspace ON messages_fts(workspace_id);
