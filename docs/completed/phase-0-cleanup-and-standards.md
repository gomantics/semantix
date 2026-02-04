# Phase 0: Cleanup & Go Open Source Standards

**Goal**: Clean up the codebase.

---

## Tasks

### 0.1 File Cleanup

Remove unused and deprecated files.

- [x] **Remove deleted bruno files** (already staged for deletion):
  - `bruno/All Workspaces.bru`
  - `bruno/Create Workspace.bru`
  - `bruno/Health.bru`
  - `bruno/Workspace Delete.bru`
  - `bruno/Workspace.bru`
  - `bruno/bruno.json`
  - `bruno/environments/Local.bru`

- [x] **Consolidate cmd directories**:
  - Removed duplicate `cmds/api/`
  - Kept only `cmd/api/main.go` (Go standard)

- [x] **Remove Milvus code** (switching to Qdrant):
  - Removed `libs/milvus/client.go`
  - Removed `libs/milvus/collection.go`
  - Updated all imports and stubbed Milvus calls with TODOs
  - Removed Milvus from docker-compose.yml
  - Removed Milvus from config

- [x] **Remove duplicate code**:
  - Removed `domains/openai/` (duplicate of `libs/openai/`)
  - Removed `domains/chunking/` (duplicate of `libs/chunking/`)

---

### 0.2 Project Structure

Follow standard Go project layout.

- [x] **Created pkg/ and internal/ directories**:
  - `pkg/pgconv/` - PostgreSQL type conversions (pgtype ↔ Go types)
  - `internal/` - Ready for application-specific code

- [x] **Added retry logic to database operations** (in `db/tx.go`):
  - `db.Query`, `db.Query1`, `db.Tx`, `db.Tx1` now have automatic retry
  - Uses `pgconn.SafeToRetry()` plus specific error codes (serialization_failure, deadlock_detected, connection errors)
  - Simple exponential backoff (10ms base, 3 attempts)

- [x] **Moved libs/ to pkg/**:
  - `libs/logger/` → `pkg/logger/`

- [x] **Moved to internal/**:
  - `api/` → `internal/api/`
  - `domains/` → `internal/domains/`

---

## Completed

Phase 0 cleanup completed. The codebase is now minimal and ready for Phase 1.

### What was removed:

**Infrastructure:**

- Milvus dependency (etcd, minio, milvus containers)
- docker-compose.yml simplified to just PostgreSQL

**Code:**

- `cmds/` directory (duplicate of `cmd/`)
- `libs/milvus/` (switching to Qdrant)
- `libs/chunking/`, `libs/openai/`, `libs/gitrepo/` (will rebuild in Phase 2)
- `domains/chunking/`, `domains/openai/` (duplicates)
- `domains/gittokens/`, `domains/repos/`, `domains/indexing/` (will rebuild)
- `api/gittokens/`, `api/repositories/`, `api/search/`, `api/workspaces/` (will rebuild)

### What remains:

```
semantix/
├── cmd/api/main.go           # Entry point
├── internal/
│   ├── api/                  # HTTP handlers
│   │   ├── run.go            # Server setup + middleware
│   │   ├── health/           # Health check endpoint
│   │   └── web/              # Context helpers
│   └── domains/
│       └── workspaces/       # Example domain (CRUD pattern)
├── pkg/
│   ├── logger/               # Zap logger setup
│   └── pgconv/               # PostgreSQL type conversions
├── config/                   # Configuration
├── db/                       # Database (sqlc generated, with retry)
├── docs/                     # Documentation
└── docker-compose.yml        # PostgreSQL only
```

The `internal/domains/workspaces/` is kept as a template showing:

- Models with JSON tags
- CRUD operations (Create, GetByID, List, Update, Delete)
- Error handling (ErrNotFound, ErrAlreadyExists)
- Pagination pattern
- Database helper usage (db.Query, db.Query1) with automatic retry
- Using pkg/pgconv for type conversions
