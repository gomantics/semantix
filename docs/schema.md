# Database Schema

> Simple PostgreSQL schema for Semantix

## Design Principles

1. **Simple types** - BIGINT for timestamps (nanoseconds since epoch), TEXT for strings
2. **No foreign keys** - Application handles referential integrity
3. **No ENUMs** - Use TEXT with application-level validation
4. **Indexes only where needed** - For actual query patterns

---

## PostgreSQL Tables

```sql
-- ============================================================================
-- WORKSPACES
-- ============================================================================
CREATE TABLE workspaces (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT,
    settings    JSONB NOT NULL DEFAULT '{}',
    created     BIGINT NOT NULL,  -- nanoseconds since epoch
    updated     BIGINT NOT NULL
);

CREATE INDEX idx_workspaces_slug ON workspaces(slug);


-- ============================================================================
-- GIT TOKENS
-- ============================================================================
CREATE TABLE git_tokens (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    provider        TEXT NOT NULL,           -- github, gitlab, bitbucket
    token_encrypted BYTEA NOT NULL,
    token_hint      TEXT,                    -- last 4 chars
    created         BIGINT NOT NULL
);

CREATE INDEX idx_git_tokens_provider ON git_tokens(provider);


-- ============================================================================
-- REPOS
-- ============================================================================
CREATE TABLE repos (
    id              BIGSERIAL PRIMARY KEY,
    workspace_id    BIGINT NOT NULL,
    git_token_id    BIGINT,                  -- nullable
    
    -- Identity
    url             TEXT NOT NULL,
    provider        TEXT NOT NULL,           -- github, gitlab, bitbucket
    owner           TEXT NOT NULL,
    name            TEXT NOT NULL,
    
    -- Queue status: pending, cloning, indexing, completed, failed
    -- Cron picks up 'pending' every 60s
    status          TEXT NOT NULL DEFAULT 'pending',
    error_message   TEXT,
    
    -- Git state (from last successful index)
    default_branch  TEXT,
    head_commit     TEXT,
    
    -- Latest stats (denormalized from last index_run)
    file_count      INT NOT NULL DEFAULT 0,
    chunk_count     INT NOT NULL DEFAULT 0,
    indexed_at      BIGINT,
    
    created         BIGINT NOT NULL,
    updated         BIGINT NOT NULL,
    
    UNIQUE(workspace_id, url)
);

CREATE INDEX idx_repos_workspace ON repos(workspace_id);
-- Queue: cron polls this
CREATE INDEX idx_repos_queue ON repos(status, created) WHERE status = 'pending';


-- ============================================================================
-- FILES
-- ============================================================================
CREATE TABLE files (
    id              BIGSERIAL PRIMARY KEY,
    repo_id         BIGINT NOT NULL,
    
    path            TEXT NOT NULL,
    content_hash    TEXT NOT NULL,           -- SHA-256 of file content
    
    language        TEXT,
    size_bytes      BIGINT NOT NULL,
    line_count      INT,
    chunk_count     INT NOT NULL DEFAULT 0,
    
    indexed_at      BIGINT,
    created         BIGINT NOT NULL,
    updated         BIGINT NOT NULL,
    
    UNIQUE(repo_id, path)
);

CREATE INDEX idx_files_repo ON files(repo_id);
CREATE INDEX idx_files_content_hash ON files(content_hash);


-- ============================================================================
-- EMBEDDING CACHE
-- ============================================================================
CREATE TABLE embedding_cache (
    content_hash    TEXT PRIMARY KEY,        -- SHA-256 of chunk content
    embedding       BYTEA NOT NULL,          -- packed float32 array
    model           TEXT NOT NULL,
    
    use_count       INT NOT NULL DEFAULT 1,
    last_used       BIGINT NOT NULL,
    created         BIGINT NOT NULL
);

CREATE INDEX idx_cache_last_used ON embedding_cache(last_used);
CREATE INDEX idx_cache_model ON embedding_cache(model);


-- ============================================================================
-- INDEX RUNS (history of each indexing attempt)
-- ============================================================================
CREATE TABLE index_runs (
    id              BIGSERIAL PRIMARY KEY,
    repo_id         BIGINT NOT NULL,
    
    -- Status: running, completed, failed
    status          TEXT NOT NULL DEFAULT 'running',
    error_message   TEXT,
    
    -- Git snapshot
    from_commit     TEXT,                    -- previous HEAD (null on first index)
    to_commit       TEXT,                    -- current HEAD we indexed
    commit_message  TEXT,                    -- message of to_commit
    branch          TEXT,
    commits_between INT NOT NULL DEFAULT 0,  -- commits skipped (from_commit..to_commit)
    
    -- Stats
    files_total     INT NOT NULL DEFAULT 0,
    files_added     INT NOT NULL DEFAULT 0,
    files_changed   INT NOT NULL DEFAULT 0,
    files_deleted   INT NOT NULL DEFAULT 0,
    chunks_created  INT NOT NULL DEFAULT 0,
    
    -- Embedding stats
    cache_hits      INT NOT NULL DEFAULT 0,
    cache_misses    INT NOT NULL DEFAULT 0,
    tokens_used     BIGINT NOT NULL DEFAULT 0,  -- actual tokens sent to OpenAI
    
    -- Timing
    started_at      BIGINT NOT NULL,
    completed_at    BIGINT,
    duration_ms     BIGINT,
    
    created         BIGINT NOT NULL
);

-- Show recent runs for a repo (for UI)
CREATE INDEX idx_runs_repo ON index_runs(repo_id, created DESC);
```

---

## Qdrant Collection

```json
{
  "collection_name": "chunks",
  "vectors_config": {
    "size": 1536,
    "distance": "Cosine"
  }
}
```

### Payload Schema

| Field | Type | Indexed | Description |
|-------|------|---------|-------------|
| `workspace_id` | integer | yes | Multi-tenant filtering |
| `repo_id` | integer | yes | Filter by repo |
| `file_id` | integer | yes | For deletion on file change |
| `file_path` | keyword | yes | Filter by path patterns |
| `language` | keyword | yes | Filter by language |
| `content` | text | no | Chunk text content |
| `content_hash` | keyword | no | SHA-256 of chunk |
| `chunk_index` | integer | no | Order within file |
| `start_line` | integer | no | Start line number |
| `end_line` | integer | no | End line number |

### Create Indexes

```go
// Required payload indexes for filtering
client.CreatePayloadIndex("chunks", "workspace_id", qdrant.PayloadSchemaType_Integer)
client.CreatePayloadIndex("chunks", "repo_id", qdrant.PayloadSchemaType_Integer)
client.CreatePayloadIndex("chunks", "file_id", qdrant.PayloadSchemaType_Integer)
client.CreatePayloadIndex("chunks", "file_path", qdrant.PayloadSchemaType_Keyword)
client.CreatePayloadIndex("chunks", "language", qdrant.PayloadSchemaType_Keyword)
```

---

## Query Patterns

### Common Queries

```sql
-- List repos in workspace
SELECT * FROM repos WHERE workspace_id = $1 ORDER BY name;

-- Get files for repo
SELECT id, path, content_hash FROM files WHERE repo_id = $1;

-- Find changed files (compare hashes)
-- App fetches all files, compares with disk hashes in memory

-- Check embedding cache (batch)
SELECT content_hash, embedding FROM embedding_cache 
WHERE content_hash = ANY($1);

-- Insert/update cache entry
INSERT INTO embedding_cache (content_hash, embedding, model, use_count, last_used, created)
VALUES ($1, $2, $3, 1, $4, $4)
ON CONFLICT (content_hash) DO UPDATE SET
    use_count = embedding_cache.use_count + 1,
    last_used = $4;

-- LRU cache eviction (delete oldest 10%)
DELETE FROM embedding_cache 
WHERE content_hash IN (
    SELECT content_hash FROM embedding_cache 
    ORDER BY last_used LIMIT $1
);
```

### Qdrant Queries

```go
// Semantic search in workspace
client.Search("chunks", vector, &qdrant.SearchParams{
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            qdrant.FieldCondition("workspace_id", qdrant.Match{Integer: workspaceID}),
        },
    },
    Limit: 10,
    WithPayload: true,
})

// Delete chunks for a file
client.Delete("chunks", &qdrant.PointsSelector{
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            qdrant.FieldCondition("file_id", qdrant.Match{Integer: fileID}),
        },
    },
})
```

---

## Data Flow

### Queue (Cron every 60s)

```go
// Cron job runs every 60 seconds
func ProcessQueue(ctx context.Context) {
    // Pick one pending repo (FIFO)
    repo := db.Query(`
        SELECT * FROM repos 
        WHERE status = 'pending' 
        ORDER BY created 
        LIMIT 1 FOR UPDATE SKIP LOCKED
    `)
    
    if repo == nil {
        return // nothing to process
    }
    
    // Mark as indexing
    db.Exec(`UPDATE repos SET status = 'indexing' WHERE id = $1`, repo.ID)
    
    // Create index run record
    runID := db.Insert(`INSERT INTO index_runs (repo_id, status, started_at, created) ...`)
    
    // Do the indexing
    err := indexRepo(ctx, repo, runID)
    
    // Update status based on result
    if err != nil {
        db.Exec(`UPDATE repos SET status = 'failed', error_message = $2 WHERE id = $1`, repo.ID, err)
        db.Exec(`UPDATE index_runs SET status = 'failed', error_message = $2 WHERE id = $1`, runID, err)
    } else {
        db.Exec(`UPDATE repos SET status = 'completed', indexed_at = $2 WHERE id = $1`, repo.ID, now)
        db.Exec(`UPDATE index_runs SET status = 'completed', completed_at = $2 WHERE id = $1`, runID, now)
    }
}
```

### Indexing

```
1. git pull (or clone if first time)
2. Create index_runs record with status = 'running'
3. Get HEAD commit (to_commit), message, branch
4. Get previous head from repos.head_commit (from_commit)
5. Count commits between: git rev-list from_commit..to_commit --count
6. Walk files, compute SHA-256 for each
7. Compare with files table:
   - added: new paths
   - changed: hash mismatch  
   - deleted: in DB but not on disk
6. For added/changed files:
   a. Chunk with tree-sitter
   b. For each chunk:
      - Hash chunk content
      - Check embedding_cache → cache_hits++
      - If miss: call OpenAI, insert to cache → cache_misses++
   c. Delete old chunks from Qdrant (by file_id)
   d. Insert new chunks to Qdrant
7. Update files table (upsert added/changed, delete removed)
8. Update index_runs with final stats
9. Update repos with latest stats from run
```

### Triggering Re-index

```sql
-- Manual re-index: just set status back to pending
UPDATE repos SET status = 'pending' WHERE id = $1;

-- Cron will pick it up within 60 seconds
```

### Search

```
1. Generate query embedding (OpenAI)
2. Search Qdrant with workspace_id filter
3. Return file_path, content, lines, score
```

---

## API Response Examples

### GET /v1/workspaces/:wid/repos/:rid

```json
{
  "id": 42,
  "url": "https://github.com/acme/api",
  "provider": "github",
  "owner": "acme",
  "name": "api",
  "status": "completed",
  "default_branch": "main",
  "head_commit": "abc1234",
  "file_count": 342,
  "chunk_count": 1205,
  "indexed_at": 1706745600000000000,
  "created": 1706659200000000000
}
```

### GET /v1/workspaces/:wid/repos/:rid/estimate

Estimate tokens before indexing.

```json
{
  "files_total": 342,
  "files_indexable": 298,
  "files_skipped": 44,
  "skipped_by_category": {
    "dependencies": 12,
    "binary": 18,
    "generated": 8,
    "lock_files": 3,
    "build": 3
  },
  "chunks_estimated": 1200,
  "tokens_total": 485000,
  "tokens_cached": 420000,
  "tokens_new": 65000
}
```

**How it works:**
1. Clone/pull repo (or use existing clone)
2. Count indexable files (skip non-indexable)
3. Chunk files, count tokens with `tiktoken-go` (`cl100k_base`)
4. Check embedding cache for existing chunk hashes
5. Return: total tokens, cached, new (what will be sent to OpenAI)

**Skipped files:**

| Category | Patterns |
|----------|----------|
| **Dependencies** | `node_modules/`, `vendor/`, `.venv/`, `venv/`, `__pycache__/` |
| **Build outputs** | `dist/`, `build/`, `out/`, `.next/`, `target/`, `bin/` |
| **Git** | `.git/` |
| **IDE** | `.idea/`, `.vscode/`, `*.swp`, `.DS_Store` |
| **Lock files** | `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `Gemfile.lock`, `poetry.lock`, `Cargo.lock` |
| **Generated** | `*.min.js`, `*.min.css`, `*.map`, `*.pb.go`, `*.generated.*` |
| **Binary** | `*.png`, `*.jpg`, `*.gif`, `*.ico`, `*.woff`, `*.ttf`, `*.pdf`, `*.zip`, `*.tar`, `*.exe`, `*.dll`, `*.so`, `*.dylib` |
| **Data** | `*.sql`, `*.csv` (large), `*.parquet`, `*.sqlite` |
| **Snapshots** | `__snapshots__/`, `*.snap` |

**Configurable via `workspaces.settings`:**
```json
{
  "exclude_patterns": ["docs/", "*.test.ts"],
  "include_patterns": ["*.go", "*.ts", "*.py"]
}
```

---

### GET /v1/workspaces/:wid/repos/:rid/runs

```json
{
  "runs": [
    {
      "id": 101,
      "status": "completed",
      "from_commit": "def5678",
      "to_commit": "abc1234",
      "commit_message": "feat: add user auth",
      "branch": "main",
      "commits_between": 9,
      "files_total": 342,
      "files_added": 5,
      "files_changed": 12,
      "files_deleted": 2,
      "chunks_created": 45,
      "cache_hits": 1160,
      "cache_misses": 45,
      "tokens_used": 18500,
      "duration_ms": 12340,
      "created": 1706745600000000000
    },
    {
      "id": 100,
      "status": "completed",
      "from_commit": null,
      "to_commit": "def5678",
      "commit_message": "fix: login bug",
      "branch": "main",
      "commits_between": 0,
      "files_total": 339,
      "files_added": 339,
      "files_changed": 0,
      "files_deleted": 0,
      "chunks_created": 1205,
      "cache_hits": 0,
      "cache_misses": 1205,
      "tokens_used": 485000,
      "duration_ms": 45200,
      "created": 1706659200000000000
    }
  ]
}
```

**UI can show:**
- "Indexed abc1234 (feat: add user auth) - skipped 9 commits"
- "First index: def5678 (fix: login bug) - 339 files"

---

## Storage Estimates

| Table | 100 repos × 10k files | Notes |
|-------|----------------------|-------|
| workspaces | < 1 KB | Negligible |
| repos | < 10 KB | Negligible |
| files | ~50 MB | 1M files × 50 bytes/row |
| embedding_cache | ~3 GB | 500k chunks × 6KB embedding |

| Qdrant | 500k chunks | Notes |
|--------|-------------|-------|
| Vectors | ~3 GB | 1536 dim × 4 bytes × 500k |
| Payloads | ~500 MB | Content + metadata |
