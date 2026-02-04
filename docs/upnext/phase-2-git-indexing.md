# Phase 2: Git & Indexing

**Goal**: Clone repositories and index code into Qdrant

---

## Overview

This phase implements the core indexing pipeline: cloning repositories, chunking code with Tree-sitter, generating embeddings, and storing vectors in Qdrant. Also includes background job processing and status tracking.

---

## Tasks

### 2.1 Git Clone with Token Auth

Implement repository cloning with support for multiple git providers.

- [ ] **Shallow clone** (`--depth 1`)
  - No commit history needed
  - Significantly reduces disk space and clone time

- [ ] **Sparse checkout** (exclude patterns):
  ```
  node_modules/
  vendor/
  .git/
  *.lock
  *.min.js
  dist/
  build/
  ```

- [ ] **Token authentication** for private repos:
  - GitHub: `https://x-access-token:{token}@github.com/...`
  - GitLab: `https://oauth2:{token}@gitlab.com/...`
  - Bitbucket: `https://x-token-auth:{token}@bitbucket.org/...`

- [ ] **Pull for updates** if repo already cloned

- [ ] **Clone path structure**: `{repos_path}/{workspace_id}/{repo_id}/`

**Files to create/modify:**
- `libs/gitrepo/client.go` - clone/pull operations
- `libs/gitrepo/provider.go` - provider-specific URL formatting
- `libs/gitrepo/github.go` - GitHub-specific handling

---

### 2.2 Multi-Provider Support

Support the major git hosting providers.

- [ ] **GitHub**
  - Personal Access Tokens (classic and fine-grained)
  - URL format: `https://github.com/{owner}/{repo}.git`

- [ ] **GitLab**
  - Personal/project access tokens
  - URL format: `https://gitlab.com/{group}/{project}.git`

- [ ] **Bitbucket**
  - App passwords
  - URL format: `https://bitbucket.org/{workspace}/{repo}.git`

- [ ] **Auto-detection** of provider from URL

**Files to create/modify:**
- `libs/gitrepo/provider.go`
- `domains/gittokens/gittokens.go` - token storage/retrieval

---

### 2.3 Tree-sitter Chunking

AST-aware code chunking using chunkx or similar.

- [ ] **Language detection** from file extension and content

- [ ] **Chunking strategy**:
  - Functions/methods as primary chunks
  - Classes/structs with their methods
  - Large functions split at logical boundaries
  - Target: ~500 tokens per chunk

- [ ] **Chunk metadata**:
  ```go
  type Chunk struct {
      Content    string
      FilePath   string
      StartLine  int
      EndLine    int
      Language   string
      ChunkType  string  // function, class, method, block
      SymbolName string  // e.g., "Login", "UserService"
  }
  ```

- [ ] **Language support** (priority order):
  - Go, Python, JavaScript/TypeScript
  - Java, Rust, C/C++
  - Ruby, PHP
  - Markdown, YAML, JSON (as text)

**Files to create/modify:**
- `libs/chunking/chunker.go`
- `domains/chunking/chunker.go` - higher-level orchestration

---

### 2.4 OpenAI Embedding Generation

Generate embeddings with batching for efficiency.

- [ ] **Model**: `text-embedding-3-small` (1536 dimensions)

- [ ] **Batching**:
  - Up to 2048 inputs per request
  - Batch by token count (max 8191 tokens per input)
  - Respect rate limits

- [ ] **Error handling**:
  - Retry with exponential backoff
  - Handle rate limit errors (429)
  - Log failed chunks for debugging

- [ ] **Input preparation**:
  - Prepend file path context
  - Format: `File: {path}\n\n{content}`

**Files to create/modify:**
- `libs/openai/embeddings.go`
- `domains/openai/embeddings.go` - domain-level wrapper

---

### 2.5 Qdrant Upsert

Store vectors with full payload for search.

- [ ] **Point structure**:
  ```go
  type Point struct {
      ID      string  // UUID
      Vector  []float32
      Payload map[string]interface{}{
          "workspace_id": "...",
          "repo_id":      "...",
          "file_id":      "...",
          "file_path":    "...",
          "language":     "...",
          "start_line":   45,
          "end_line":     78,
          "chunk_content": "...",
          "chunk_type":   "function",
          "symbol_name":  "Login",
      }
  }
  ```

- [ ] **Batch upsert** (100-500 points per request)

- [ ] **Point ID strategy**: UUID per chunk, deterministic from content hash

**Files to create/modify:**
- `libs/qdrant/collection.go` - add Upsert method

---

### 2.6 Background Job Queue

Simple cron-based queue for indexing jobs.

- [ ] **Polling mechanism**:
  - Check `repos` table every 60 seconds
  - Find repos with `status = 'pending'`
  - Process one at a time (or up to `max_workers`)

- [ ] **Status transitions**:
  ```
  pending → indexing → ready
                    ↘ error
  ```

- [ ] **Concurrency control**:
  - `max_workers` config (default: 4)
  - Use worker pool pattern

- [ ] **Graceful shutdown**:
  - Finish current job before exit
  - Don't start new jobs during shutdown

**Files to create/modify:**
- `domains/indexing/orchestrator.go` - job scheduling
- `domains/indexing/worker.go` - individual job processing

---

### 2.7 Index Runs Table

Track indexing history for debugging and UI.

- [ ] **Schema**:
  ```sql
  CREATE TABLE index_runs (
      id UUID PRIMARY KEY,
      repo_id UUID NOT NULL,
      status VARCHAR(20) NOT NULL,  -- running, completed, failed
      started_at BIGINT NOT NULL,
      completed_at BIGINT,
      files_processed INT DEFAULT 0,
      chunks_created INT DEFAULT 0,
      embeddings_generated INT DEFAULT 0,
      embeddings_cached INT DEFAULT 0,
      error_message TEXT,
      duration_ms BIGINT
  );
  ```

- [ ] **Update during processing**:
  - Increment counters as files are processed
  - Record final stats on completion

- [ ] **Retention**: Keep last N runs per repo

**Files to create/modify:**
- `db/schema/index_runs.sql`
- `db/queries/index_runs.sql`

---

### 2.8 Status Tracking & Error Handling

Robust status management and error reporting.

- [ ] **Repo status enum**:
  - `pending` - waiting to be indexed
  - `cloning` - git clone in progress
  - `indexing` - processing files
  - `ready` - indexing complete
  - `error` - indexing failed

- [ ] **Error categories**:
  - Clone errors (auth, network, not found)
  - Parse errors (chunking failed)
  - Embedding errors (API failures)
  - Storage errors (Qdrant/Postgres issues)

- [ ] **Error storage**:
  - Store in `repos.error_message`
  - Include actionable context
  - Store in `index_runs` for history

- [ ] **Retry logic**:
  - Automatic retry for transient errors
  - Max retries with backoff
  - Mark as error after max retries

**Files to create/modify:**
- `domains/repos/status.go`
- `domains/indexing/worker.go`

---

## Indexing Pipeline Flow

```
POST /v1/workspaces/:wid/repos  (status = pending)
          │
          ▼
    ┌─────────────┐
    │ Orchestrator│  ← Polls every 60s for pending repos
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │   Worker    │  ← Claims job, sets status = cloning
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ Git Clone   │  ← Shallow clone with token auth
    └──────┬──────┘
           │ status = indexing
           ▼
    ┌─────────────┐
    │ Walk Files  │  ← Hash each file, filter by extension
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ Chunk Files │  ← Tree-sitter AST chunking
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ Embed Batch │  ← OpenAI API (batched)
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ Qdrant      │  ← Upsert points with payload
    │ Upsert      │
    └──────┬──────┘
           │
           ▼
    ┌─────────────┐
    │ Update DB   │  ← files table, repo status = ready
    └─────────────┘
```

---

## Acceptance Criteria

- [ ] Can clone public repositories without auth
- [ ] Can clone private repositories with valid token
- [ ] Files are chunked with correct line numbers
- [ ] Embeddings are generated and stored in Qdrant
- [ ] Qdrant points have correct payload metadata
- [ ] Index runs are recorded with accurate stats
- [ ] Failed indexing sets repo to error status with message
- [ ] Background worker processes pending repos automatically

---

## Dependencies

- `github.com/go-git/go-git/v5` - Git operations
- Tree-sitter Go bindings or `chunkx` CLI
- `github.com/sashabaranov/go-openai` - OpenAI client
- `github.com/qdrant/go-client` - Qdrant client

---

## Notes

- Consider implementing a simple file walker first, add Tree-sitter chunking after basic flow works
- Embedding costs: ~$0.02 per 1M tokens for `text-embedding-3-small`
- Large repos (>10k files) may need streaming/pagination
- Set reasonable timeouts for git clone operations (10-15 minutes for large repos)
