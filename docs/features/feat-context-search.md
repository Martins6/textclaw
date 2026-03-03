# Overview

Implements Context Search (Historical Memory) feature that allows workspace containers to search conversation history using vector similarity search, keyword search, and recent message retrieval.

# Details

- Add sqlite-vec for vector storage and KNN similarity search
- Add gollama.cpp (PureGo) for local embedding generation with llama.cpp
- Download nomic-embed-text-v1.5-GGUF embedding model (768 dimensions)
- Create database migrations for vector embeddings table and FTS5 virtual table
- Add database query functions: SaveMessageEmbedding, SearchBySimilarity, SearchByKeyword, GetRecentMessages
- Implement Embedding Service to load GGUF model and generate embeddings
- Add Context Search Socket Handler for container→daemon communication
- Add Context Search CLI client and commands (similar, search, recent, find)
- Lazy load embedding model on first use to minimize memory usage

# File Paths

- internal/database/migrations/003_vector_embeddings.sql
- internal/database/queries.go
- internal/embedding/service.go
- internal/daemon/context/search.go
- pkg/socket/client.go
- internal/cli/context.go
- cmd/textclaw/main.go
