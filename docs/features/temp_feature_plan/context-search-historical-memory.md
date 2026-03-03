# Context Search (Historical Memory) Implementation Plan

## Problem

The TextClaw system needs a Context Search (Historical Memory) feature that allows workspace containers to search conversation history for context - finding previous messages that are semantically similar, keyword matches, or recent conversations.

## Solution Overview

Implement a socket-based context search system using:
- **sqlite-vec** for vector storage and similarity search (KNN)
- **llama.cpp** with Go bindings for local embedding generation
- Existing Unix socket infrastructure for secure container→daemon communication

## Embedding Stack Details

| Component | Technology | Details |
|-----------|------------|---------|
| **Vector Storage** | `sqlite-vec` | SQLite extension, ~30MB, works locally |
| **Embedding Model** | llama.cpp | Go bindings (gollama.cpp - PureGo, no CGO) |
| **Embedding Model File** | nomic-embed-text-v1.5-GGUF | 768 dimensions, ~274MB (Q8), available on HuggingFace |

## Steps to Solve

### Step 1: Add Dependencies
- Add sqlite-vec Go bindings to go.mod
- Add gollama.cpp (PureGo, no CGO required)

### Step 2: Download Embedding Model
- Download nomic-embed-text-v1.5-GGUF from HuggingFace
- Put in TextClaw models directory

### Step 3: Add Database Schema for Context Search
- Create migration for vector embeddings table (sqlite-vec)
- Create FTS5 virtual table for keyword search
- Add workspace isolation indexes

### Step 4: Add Database Query Functions
- SaveMessageEmbedding(workspaceID, messageID, embedding)
- SearchBySimilarity(workspaceID, embedding, limit) - sqlite-vec KNN
- SearchByKeyword(workspaceID, query, limit) - FTS5
- GetRecentMessages(workspaceID, limit)

### Step 5: Add Embedding Service (llama.cpp)
- Load GGUF model with embeddings option
- GenerateEmbedding(text) function
- Model lifecycle management

### Step 6: Add Context Search Socket Handler
- Handle CONTEXT_SEARCH socket messages
- Route to embedding service or FTS based on search type
- Return formatted results

### Step 7: Add Context Search CLI Client
- Add ContextSearch() function to socket client

### Step 8: Add Context CLI Commands
- Add textclaw context subcommand with:
  - similar <query> - semantic search
  - search <query> - keyword search
  - recent [limit] - recent messages
  - find <query> - full-text search

### Step 9: Register Context Command in CLI
- Add context command to CLI root

## Embedding Generation Flow

1. User: textclaw context similar "what did I say about python"
2. CLI reads workspace_id from ~/.textclaw.json
3. CLI sends: "CONTEXT_SEARCH|semantic|query|limit"
4. Daemon loads GGUF model (if not loaded) - nomic-embed-text-v1.5
5. Daemon generates embedding: "search_query: {text}"
6. llama.cpp returns: [0.123, -0.456, ...] (768-dim)
7. Daemon queries sqlite-vec with embedding + workspace_id filter
8. Return matching messages

## Challenges

- llama.cpp model loading: Initial load can be slow (~5-10 seconds)
  - Mitigation: Keep model loaded in memory, lazy load on first use
- Embedding dimension mismatch: Must use 768 dimensions
  - Mitigation: Validate on insert/query

## Risk Assessment

- **Level:** Medium
- **Key Risks:**
  - Model file download size (~274MB for Q8)
  - First-time embedding generation latency

## Fallback Plan

- If llama.cpp fails: Use keyword search only
- If sqlite-vec unavailable: Use basic LIKE queries
- If both unavailable: Return error with setup instructions
