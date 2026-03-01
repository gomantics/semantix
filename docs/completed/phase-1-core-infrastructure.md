# Phase 1: Core Infrastructure

**Goal**: Basic working system with Qdrant + PostgreSQL

---

## Overview

This phase establishes the foundational infrastructure: database schemas, vector store setup, configuration management, and containerized development environment.

---

## Tasks

### 1.1 PostgreSQL Schema

Set up the core relational schema for metadata storage.

- [x] **Workspaces table**
  - `id` (UUID, primary key)
  - `name` (string, unique)
  - `created_at`, `updated_at` (timestamps)

- [x] **Repositories table**
  - `id` (UUID, primary key)
  - `workspace_id` (UUID, foreign key)
  - `url` (string, git URL)
  - `branch` (string, default: main)
  - `status` (enum: pending, indexing, ready, error)
  - `indexed_at` (timestamp, nullable)
  - `error_message` (text, nullable)
  - `created_at`, `updated_at` (timestamps)

- [x] **Files table**
  - `id` (UUID, primary key)
  - `repo_id` (UUID, foreign key)
  - `path` (string, relative path)
  - `content_hash` (string, SHA-256)
  - `size_bytes` (bigint)
  - `language` (string, detected)
  - `indexed_at` (timestamp)

- [x] **Git Tokens table**
  - `id` (UUID, primary key)
  - `name` (string, display name)
  - `provider` (enum: github, gitlab, bitbucket)
  - `token_encrypted` (bytea, encrypted PAT)
  - `created_at` (timestamp)

**Files to create/modify:**
- `internal/db/schema/workspaces.sql`
- `internal/db/schema/repos.sql`
- `internal/db/schema/files.sql`
- `internal/db/schema/git_tokens.sql`
- `internal/db/queries/*.sql` (SQLC queries)

---

### 1.2 Qdrant Collection Setup

Configure Qdrant for vector storage with proper indexing.

- [x] **Create collection** with:
  - Vector size: 1536 (OpenAI `text-embedding-3-small`)
  - Distance metric: Cosine
  - On-disk storage enabled

- [x] **Payload schema** (stored with each vector):
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

- [x] **Create payload indexes** for:
  - `workspace_id` (integer) - required for multi-tenancy
  - `repo_id` (integer) - filter by repo
  - `file_id` (integer) - filter by file
  - `language` (keyword) - filter by language
  - `file_path` (keyword) - path pattern matching

**Files to create/modify:**
- `internal/qdrant/init.go` - connection management
- `internal/qdrant/collection.go` - collection setup, upsert, search

---

### 1.3 Configuration Management

Implement hierarchical config: defaults -> TOML file -> environment variables.

- [x] **Config struct** with sections:
  - `Server` (port, host)
  - `Database` (DSN)
  - `Qdrant` (address, collection name)
  - `OpenAI` (API key)
  - `Indexing` (workers, chunk size, cache settings)

- [x] **Loading order**:
  1. Embedded defaults
  2. `config.toml` file (optional)
  3. Environment variables (e.g., `CONFIG_DATABASE_DSN`)

- [x] **Validation** on startup

**Files to create/modify:**
- `config/config.go` - struct definitions
- `config/config.toml` - default configuration
- `config/config.gen.go` - generated helpers

---

### 1.4 Docker Compose

Development environment with all dependencies.

- [x] **PostgreSQL 17** container
  - Volume for data persistence
  - Health check
  - Default credentials for dev

- [x] **Qdrant** container
  - Volume for storage
  - Expose REST (6333) and gRPC (6334) ports
  - Health check

- [x] **Network** for service communication (implicit default)

**Files to create/modify:**
- `docker-compose.yml`

---

### 1.5 Health Check Endpoint

Basic health endpoint for infrastructure verification.

- [x] **GET /v1/health** returns:
  ```json
  {
    "status": "ok",
    "database": "ok",
    "qdrant": "ok",
    "version": "0.1.0"
  }
  ```

- [x] Check actual connectivity to Postgres and Qdrant
- [x] Return appropriate status codes (200 OK, 503 Service Unavailable)

**Files to create/modify:**
- `internal/api/health/get.go`
- `internal/api/health/router.go`

---

## Acceptance Criteria

- [x] `docker-compose up` starts Postgres and Qdrant successfully
- [x] Application connects to both services on startup
- [x] `/v1/health` returns healthy status with connected services
- [x] Database migrations apply cleanly
- [x] Qdrant collection is created with correct schema

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
- IDs use BIGSERIAL (not UUID) for performance; workspace_id etc. are BIGINT
