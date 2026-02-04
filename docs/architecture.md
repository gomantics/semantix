# Semantix Architecture

> Semantic code search engine inspired by [Cursor's secure codebase indexing](https://cursor.com/blog/secure-codebase-indexing)

## Overview

Semantix is an open-source semantic code search service for organizations. Teams add their repositories once, and Semantix provides:

1. **Search API** - Semantic search over all indexed codebases
2. **MCP Server** - AI assistants (Cursor, Claude, etc.) can search and understand your code
3. **Documentation** (future) - Auto-generated docs per repo and workspace

## Target Users

- Engineering teams wanting semantic search across their codebase
- Organizations using AI coding assistants that need codebase context
- Self-hosted deployments with private repositories

## Design Decisions

### Why Clone Repositories Locally?

For "set up once, search forever" use case, cloning locally is the right approach:

| Approach                   | Trade-off                                                |
| -------------------------- | -------------------------------------------------------- |
| **Clone locally** ✓        | Disk space (~100MB/repo), but fast access and simple     |
| Git provider API           | Rate limited (5000 req/hr GitHub), complex               |
| Client-side (Cursor model) | Requires IDE plugin, can't index repos user doesn't have |

**Optimizations:**

- Shallow clone (`--depth 1`) - no history needed
- Sparse checkout - skip `node_modules`, `vendor`, binaries
- Webhooks trigger incremental re-index on push

---

## Vector Database: Qdrant

We use **Qdrant** over Milvus for simpler deployment and better developer experience:

| Why Qdrant               | Details                                            |
| ------------------------ | -------------------------------------------------- |
| **Simple deployment**    | Single container (vs Milvus needing etcd + MinIO)  |
| **Lower resource usage** | No coordination overhead                           |
| **Payload filtering**    | Rich filtering on metadata alongside vector search |
| **Secondary indexes**    | Efficient filtering without vector scans           |
| **Rust performance**     | Memory safety with zero-cost abstractions          |
| **Good Go client**       | `github.com/qdrant/go-client`                      |

**Trade-offs:** Slightly smaller community than Milvus, no built-in partition keys (use payload filtering instead)

---

## Target Architecture

### Key Concepts

1. **Content Hashing**: SHA-256 each file to detect changes on re-index
2. **Embedding Cache**: Hash chunk content → reuse embeddings (global, not per-repo)
3. **Incremental Sync**: Only process files where hash changed
4. **Similarity Hash (future)**: SimHash to bootstrap from similar codebases

### System Components

```
                              ┌─────────────────┐
                              │   AI Assistants │
                              │ (Cursor, Claude)│
                              └────────┬────────┘
                                       │ MCP Protocol
                                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Semantix Server                          │
├────────────────────────────────┬────────────────────────────────┤
│          REST API              │          MCP Server            │
│  - Workspaces                  │  - search (semantic query)     │
│  - Repositories                │  - get_file (retrieve code)    │
│  - Search                      │  - list_repos                  │
│  - Git Tokens                  │  - get_context                 │
└──────────────┬─────────────────┴───────────────┬────────────────┘
               │                                 │
               ▼                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Domain Services                           │
├─────────────────────────────────────────────────────────────────┤
│  Indexing    │  Chunking  │  Embeddings  │  Git Operations      │
│  Orchestrator│  (chunkx)  │  (OpenAI)    │  (clone/pull)        │
└──────┬───────┴─────┬──────┴──────┬───────┴────────┬─────────────┘
       │             │             │                │
       ▼             ▼             ▼                ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Data Layer                               │
├──────────────────────┬──────────────────────────────────────────┤
│     PostgreSQL       │              Qdrant                      │
│  - Workspaces        │  - Vector embeddings                     │
│  - Repositories      │  - Chunk content + metadata              │
│  - Files (metadata)  │  - Similarity search                     │
│  - Git Tokens        │                                          │
│  - Embedding Cache   │                                          │
└──────────────────────┴──────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Cloned Repositories                         │
│                    (shallow, sparse checkout)                   │
└─────────────────────────────────────────────────────────────────┘
```

### MCP Server Tools

The MCP (Model Context Protocol) server exposes tools for AI assistants:

| Tool          | Description                                              |
| ------------- | -------------------------------------------------------- |
| `search`      | Semantic search: "how does authentication work?"         |
| `get_file`    | Retrieve file contents by path                           |
| `list_repos`  | List indexed repositories in workspace                   |
| `get_context` | Get relevant code snippets for a query (search + expand) |
| `get_symbols` | List functions/classes in a file (future)                |

### Data Models

See [schema.md](./schema.md) for the complete database schema.

**Key design decisions:**

1. **Files table as hash map** - Each file has `content_hash`, compare on re-index to find changes
2. **No merkle tree storage** - Just compare file hashes directly (O(n) but n is small and fast)
3. **Content-addressed embedding cache** - SHA-256 of chunk content → cached embedding (global)
4. **Row-level multi-tenancy** - `workspace_id` filtering, not schema-per-tenant
5. **Qdrant for chunks** - Vectors + payload stored together, PostgreSQL only has file metadata
6. **Simple types** - BIGINT for timestamps (nanoseconds), no foreign keys

### Indexing Pipeline

```
Repository URL
     │
     ▼
┌─────────────────┐
│  Clone / Pull   │  ← Shallow clone, sparse checkout
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Hash All Files │  ← SHA-256 each file, store in files table
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Compare Hashes │  ← Query files table, find changed/new/deleted
│  with DB        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Chunk Changed  │  ← Tree-sitter AST-aware chunking
│  Files          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Check Cache    │  ← SHA-256(chunk_content) → cached embedding?
│  for Embeddings │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
 Cached    Not Cached
    │         │
    │         ▼
    │    ┌─────────────────┐
    │    │  Generate       │  ← OpenAI API (batched)
    │    │  Embeddings     │
    │    └────────┬────────┘
    │             │
    │             ▼
    │    ┌─────────────────┐
    │    │  Store in       │
    │    │  Cache          │
    │    └────────┬────────┘
    │             │
    └──────┬──────┘
           │
           ▼
┌─────────────────┐
│  Upsert to      │  ← Delete old chunks for changed files
│  Qdrant         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Update Files   │  ← Update hashes, indexed_at in DB
│  Table          │
└─────────────────┘
```

**Change detection is simple:**

- Files table stores `content_hash` (SHA-256) for each file
- On re-index: hash files on disk, compare to DB
- Changed = hash mismatch, New = not in DB, Deleted = in DB but not on disk
- No separate merkle tree needed - files table IS the hash map

### API Endpoints

Resources are organized hierarchically under workspaces for clear scoping and security.

```
# Global (no workspace context)
GET    /v1/health                              # Health check
GET    /v1/stats                               # Aggregate index statistics

# Git Tokens (org/user level, shared across workspaces)
GET    /v1/gittokens                           # List tokens (masked)
POST   /v1/gittokens                           # Add git token
DELETE /v1/gittokens/:id                       # Remove token

# Workspaces
GET    /v1/workspaces                          # List workspaces
POST   /v1/workspaces                          # Create workspace
GET    /v1/workspaces/:wid                     # Get workspace details
DELETE /v1/workspaces/:wid                     # Delete workspace
GET    /v1/workspaces/:wid/stats               # Workspace-specific stats

# Repositories (workspace-scoped)
GET    /v1/workspaces/:wid/repos               # List repos in workspace
POST   /v1/workspaces/:wid/repos               # Add repository to index
GET    /v1/workspaces/:wid/repos/:rid          # Get repo status
DELETE /v1/workspaces/:wid/repos/:rid          # Remove repo from workspace
POST   /v1/workspaces/:wid/repos/:rid/reindex  # Trigger re-index (sets status=pending)
GET    /v1/workspaces/:wid/repos/:rid/runs     # Index history for UI
GET    /v1/workspaces/:wid/repos/:rid/estimate # Estimate tokens for indexing

# Search (workspace-scoped)
POST   /v1/workspaces/:wid/search              # Semantic search
POST   /v1/workspaces/:wid/search/hybrid       # Hybrid semantic + keyword search
```

**Design rationale:**

- **Nested routes**: Workspace ID in path enables middleware-level authorization
- **Repos are workspace-scoped**: The same GitHub repo can be indexed in multiple workspaces with different configurations
- **Git tokens are global**: Credentials are org/user-level, reusable across workspaces
- **Search under workspace**: Clear scoping, no risk of cross-workspace data leaks

---

## Implementation Phases

### Phase 1: Core Infrastructure

**Goal**: Basic working system with Qdrant + PostgreSQL

- [ ] PostgreSQL schema (workspaces, repos, files, git_tokens)
- [ ] Qdrant collection setup with payload indexes
- [ ] Config management (TOML + env vars)
- [ ] Docker Compose (postgres + qdrant)
- [ ] Health check endpoint

### Phase 2: Git & Indexing

**Goal**: Clone repos and index code into Qdrant

- [ ] Git clone (shallow + sparse) with token auth
- [ ] Support GitHub, GitLab, Bitbucket
- [ ] Tree-sitter chunking (`chunkx`)
- [ ] OpenAI embedding generation (batched)
- [ ] Qdrant upsert with full payload
- [ ] Cron-based queue (polls `repos` every 60s)
- [ ] `index_runs` table for history/metadata
- [ ] Status tracking and error handling

### Phase 3: Search API

**Goal**: Semantic search over indexed code

- [ ] Query embedding generation
- [ ] Qdrant vector search with workspace filtering
- [ ] Result formatting (file path, lines, content, score)
- [ ] Search filters (repo, language, path patterns)

### Phase 4: Embedding Cache

**Goal**: Avoid regenerating embeddings for unchanged content

- [ ] `embedding_cache` table
- [ ] Content hashing (SHA-256 of chunk content)
- [ ] Cache lookup before OpenAI calls
- [ ] LRU eviction job
- [ ] Cache hit/miss metrics

### Phase 5: Incremental Indexing

**Goal**: Only process changed files on re-index

- [ ] Hash files on disk, compare to `files.content_hash`
- [ ] Detect: new files, changed files, deleted files
- [ ] Delete old chunks from Qdrant for changed/deleted files
- [ ] Only chunk and embed changed/new files
- [ ] Progress tracking

### Phase 6: Periodic Refresh

**Goal**: Keep indexes fresh

- [ ] Add `refresh_interval` to repos (e.g., daily, hourly)
- [ ] Cron checks: if `indexed_at + refresh_interval < now`, set status = 'pending'
- [ ] Optional: webhook endpoints for immediate re-index

### Phase 7: MCP Server

**Goal**: AI assistants can search and understand code

- [ ] MCP protocol implementation (stdio transport)
- [ ] `search` tool - semantic search
- [ ] `get_file` tool - retrieve file contents
- [ ] `list_repos` tool - list indexed repos
- [ ] `get_context` tool - search + expand surrounding code
- [ ] Workspace authentication/scoping

### Phase 8: Documentation (Future)

**Goal**: Auto-generated docs per repo/workspace

- [ ] Per-repo documentation generation
- [ ] Architecture overview extraction
- [ ] API/function documentation
- [ ] Cross-repo relationship mapping
- [ ] Workspace-level documentation portal

### Future Enhancements

- [ ] **Hybrid search**: Semantic + keyword (BM25 or Qdrant sparse vectors)
- [ ] **Workspace index sharing**: SimHash to bootstrap from similar indexes
- [ ] **Local embeddings**: Fallback to `all-MiniLM-L6-v2` when no OpenAI key
- [ ] **Code navigation**: Go-to-definition, find references via LSP data

---

## Configuration

```toml
[server]
port = 8080

[postgres]
dsn = "postgres://semantix:semantix@localhost:5432/semantix?sslmode=disable"

[qdrant]
address = "localhost:6334"
collection_name = "semantix_chunks"
# Optional: API key for Qdrant Cloud
api_key = ""

[openai]
api_key = "${OPENAI_API_KEY}"
embedding_model = "text-embedding-3-small"

[indexing]
# Maximum concurrent indexing jobs
max_workers = 4
# Chunk size in tokens (approximate)
chunk_size = 500
# Enable embedding cache
cache_enabled = true
# Cache TTL in days (0 = never expire)
cache_ttl_days = 30

[git]
# Base path for cloned repositories
repos_path = "./data/repos"
```

---

## Deployment

### Minimal Self-Hosted (Docker Compose)

```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: semantix
      POSTGRES_PASSWORD: semantix
      POSTGRES_DB: semantix
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  qdrant:
    image: qdrant/qdrant:latest
    volumes:
      - ./data/qdrant:/qdrant/storage
    ports:
      - "6333:6333" # REST API
      - "6334:6334" # gRPC

  semantix:
    build: .
    environment:
      SEMANTIX_POSTGRES_DSN: postgres://semantix:semantix@postgres:5432/semantix?sslmode=disable
      SEMANTIX_QDRANT_ADDRESS: qdrant:6334
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    ports:
      - "8080:8080"
    volumes:
      - ./data/repos:/app/data/repos
    depends_on:
      - postgres
      - qdrant
```

### Production Considerations

1. **Qdrant**: Enable snapshots, configure replicas for HA
2. **PostgreSQL**: Use managed service or set up replication
3. **OpenAI**: Consider rate limiting, fallback to local embeddings (e.g., `all-MiniLM-L6-v2`)
4. **Secrets**: Use proper secret management (Vault, AWS Secrets Manager)
5. **Monitoring**: Add Prometheus metrics, structured logging

---

## References

- [Cursor: Securely indexing large codebases](https://cursor.com/blog/secure-codebase-indexing)
- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [Tree-sitter](https://tree-sitter.github.io/)
