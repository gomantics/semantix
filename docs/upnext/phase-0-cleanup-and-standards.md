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

### 0.2 Project Structure (Deferred)

Follow standard Go project layout. This will be done incrementally during later phases.

- [ ] **Standard directories** (to be done in Phase 1-2):

  ```
  semantix/
  ├── cmd/
  │   └── api/              # Main application entry point
  │       └── main.go
  ├── internal/             # Private application code (not importable)
  │   ├── api/              # HTTP handlers
  │   ├── domain/           # Business logic
  │   └── repository/       # Data access
  ├── pkg/                  # Public libraries (importable by others)
  │   ├── chunking/
  │   ├── gitrepo/
  │   └── qdrant/
  ├── config/               # Configuration
  ├── db/                   # Database schemas and queries
  ├── docs/                 # Documentation
  └── scripts/              # Build and utility scripts
  ```

- [ ] **Decide on internal vs pkg** (defer to Phase 1):
  - `internal/` for application-specific code
  - `pkg/` only if libraries should be importable

- [ ] **Rename directories** (defer to Phase 1):
  - `domains/` → `internal/domain/`
  - `api/` → `internal/api/`
  - `libs/` → `pkg/` or `internal/pkg/`

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
├── api/
│   ├── run.go                # Server setup + middleware
│   ├── health/               # Health check endpoint
│   └── web/                  # Context helpers
├── domains/
│   └── workspaces/           # Example domain (CRUD pattern)
├── libs/
│   └── logger/               # Zap logger setup
├── config/                   # Configuration
├── db/                       # Database (sqlc generated)
├── docs/                     # Documentation
└── docker-compose.yml        # PostgreSQL only
```

The `domains/workspaces/` is kept as a template showing:
- Models with JSON tags
- CRUD operations (Create, GetByID, List, Update, Delete)
- Error handling (ErrNotFound, ErrAlreadyExists)
- Pagination pattern
- Database helper usage (db.Query, db.Query1)
