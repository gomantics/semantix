# Phase 1: Core Infrastructure

**Goal**: Basic working system with Qdrant + PostgreSQL

---

## Overview

This phase establishes the foundational infrastructure: database schemas, vector store setup, configuration management, and containerized development environment.

---

## Tasks

### 1.1 PostgreSQL Schema

Set up the core relational schema for metadata storage.

- [ ] **Workspaces table**
  - `id` (UUID, primary key)
  - `name` (string, unique)
  - `created_at`, `updated_at` (timestamps)

- [ ] **Repositories table**
  - `id` (UUID, primary key)
  - `workspace_id` (UUID, foreign key)
  - `url` (string, git URL)
  - `branch` (string, default: main)
  - `status` (enum: pending, indexing, ready, error)
  - `indexed_at` (timestamp, nullable)
  - `error_message` (text, nullable)
  - `created_at`, `updated_at` (timestamps)

- [ ] **Files table**
  - `id` (UUID, primary key)
  - `repo_id` (UUID, foreign key)
  - `path` (string, relative path)
  - `content_hash` (string, SHA-256)
  - `size_bytes` (bigint)
  - `language` (string, detected)
  - `indexed_at` (timestamp)

- [ ] **Git Tokens table**
  - `id` (UUID, primary key)
  - `name` (string, display name)
  - `provider` (enum: github, gitlab, bitbucket)
  - `token_encrypted` (bytea, encrypted PAT)
  - `created_at` (timestamp)

**Files to create/modify:**
- `db/schema/workspaces.sql`
- `db/schema/repos.sql`
- `db/schema/files.sql`
- `db/schema/git_tokens.sql`
- `db/queries/*.sql` (SQLC queries)

---

### 1.2 Qdrant Collection Setup

Configure Qdrant for vector storage with proper indexing.

- [ ] **Create collection** with:
  - Vector size: 1536 (OpenAI `text-embedding-3-small`)
  - Distance metric: Cosine
  - On-disk storage enabled

- [ ] **Payload schema** (stored with each vector):
  ```json
  {
    "workspace_id": "uuid",
    "repo_id": "uuid",
    "file_id": "uuid",
    "file_path": "src/auth/login.go",
    "language": "go",
    "start_line": 45,
    "end_line": 78,
    "chunk_content": "func Login(...) { ... }",
    "chunk_type": "function",
    "symbol_name": "Login"
  }
  ```

- [ ] **Create payload indexes** for:
  - `workspace_id` (keyword) - required for multi-tenancy
  - `repo_id` (keyword) - filter by repo
  - `language` (keyword) - filter by language
  - `file_path` (keyword) - path pattern matching

**Files to create/modify:**
- `libs/qdrant/client.go` - connection management
- `libs/qdrant/collection.go` - collection setup, upsert, search

---

### 1.3 Configuration Management

Implement hierarchical config: defaults → TOML file → environment variables.

- [ ] **Config struct** with sections:
  - `Server` (port, host)
  - `Postgres` (DSN)
  - `Qdrant` (address, collection name, API key)
  - `OpenAI` (API key, model)
  - `Indexing` (workers, chunk size, cache settings)
  - `Git` (repos path)

- [ ] **Loading order**:
  1. Embedded defaults
  2. `config.toml` file (optional)
  3. Environment variables (e.g., `SEMANTIX_POSTGRES_DSN`)

- [ ] **Validation** on startup

**Files to create/modify:**
- `config/config.go` - struct definitions
- `config/config.toml` - default configuration
- `config/config.gen.go` - generated helpers (if using code gen)

---

### 1.4 Docker Compose

Development environment with all dependencies.

- [ ] **PostgreSQL 17** container
  - Volume for data persistence
  - Health check
  - Default credentials for dev

- [ ] **Qdrant** container
  - Volume for storage
  - Expose REST (6333) and gRPC (6334) ports
  - Health check

- [ ] **Network** for service communication

- [ ] **Optional: pgAdmin** for database UI

**Files to create/modify:**
- `docker-compose.yml`

---

### 1.5 Health Check Endpoint

Basic health endpoint for infrastructure verification.

- [ ] **GET /v1/health** returns:
  ```json
  {
    "status": "healthy",
    "postgres": "connected",
    "qdrant": "connected",
    "version": "0.1.0"
  }
  ```

- [ ] Check actual connectivity to Postgres and Qdrant
- [ ] Return appropriate status codes (200 OK, 503 Service Unavailable)

**Files to create/modify:**
- `api/health/get.go`
- `api/health/router.go`

---

## Acceptance Criteria

- [ ] `docker-compose up` starts Postgres and Qdrant successfully
- [ ] Application connects to both services on startup
- [ ] `/v1/health` returns healthy status with connected services
- [ ] Database migrations apply cleanly
- [ ] Qdrant collection is created with correct schema

---

## Dependencies

- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/qdrant/go-client` - Qdrant Go client
- `github.com/knadh/koanf/v2` - Configuration management
- `github.com/go-chi/chi/v5` - HTTP router

---

## Notes

- Use BIGINT for timestamps (nanoseconds since epoch) for consistency
- No foreign keys in PostgreSQL - handle referential integrity in application
- Qdrant collection name should be configurable for multi-environment support
