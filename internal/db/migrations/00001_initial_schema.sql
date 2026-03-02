-- Migration: 00001_initial_schema
-- Safe to re-run: yes (all statements use IF NOT EXISTS)

-- +goose Up
CREATE TABLE IF NOT EXISTS users (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created       BIGINT NOT NULL, -- nanoseconds since epoch
  updated       BIGINT NOT NULL  -- nanoseconds since epoch
);

CREATE TABLE IF NOT EXISTS sessions (
  id         BIGSERIAL PRIMARY KEY,
  user_id    BIGINT NOT NULL,
  token      TEXT NOT NULL UNIQUE, -- 32 random bytes, hex-encoded (64 chars)
  created    BIGINT NOT NULL,      -- nanoseconds since epoch
  expires_at BIGINT                -- nanoseconds since epoch, NULL = never expires
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

CREATE TABLE IF NOT EXISTS workspaces (
  id          BIGSERIAL PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT,
  settings    JSONB NOT NULL,
  created     BIGINT NOT NULL,  -- nanoseconds since epoch
  updated     BIGINT NOT NULL   -- nanoseconds since epoch
);

CREATE TABLE IF NOT EXISTS git_tokens (
  id              BIGSERIAL PRIMARY KEY,
  name            TEXT NOT NULL,
  provider        TEXT NOT NULL, -- github, gitlab, bitbucket
  token_encrypted BYTEA NOT NULL,
  token_hint      TEXT,          -- last 4 chars of token
  created         BIGINT NOT NULL -- nanoseconds since epoch
);

CREATE TABLE IF NOT EXISTS repos (
  id            BIGSERIAL PRIMARY KEY,
  workspace_id  BIGINT NOT NULL,
  git_token_id  BIGINT,         -- optional: explicit token to use for cloning
  url           TEXT NOT NULL,
  branch        TEXT NOT NULL,
  is_private    BOOLEAN NOT NULL DEFAULT false,
  status        TEXT NOT NULL,  -- pending, cloning, indexing, ready, error
  indexed_at    BIGINT,         -- nanoseconds since epoch, null until first index
  error_message TEXT,
  created       BIGINT NOT NULL, -- nanoseconds since epoch
  updated       BIGINT NOT NULL  -- nanoseconds since epoch
);

CREATE INDEX IF NOT EXISTS idx_repos_workspace_id ON repos(workspace_id);
CREATE INDEX IF NOT EXISTS idx_repos_status ON repos(status);

CREATE TABLE IF NOT EXISTS files (
  id           BIGSERIAL PRIMARY KEY,
  repo_id      BIGINT NOT NULL,
  path         TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  size_bytes   BIGINT NOT NULL,
  language     TEXT,
  indexed_at   BIGINT NOT NULL -- nanoseconds since epoch
);

CREATE INDEX IF NOT EXISTS idx_files_repo_id ON files(repo_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_repo_path ON files(repo_id, path);

CREATE TABLE IF NOT EXISTS index_runs (
  id                   BIGSERIAL PRIMARY KEY,
  repo_id              BIGINT NOT NULL,
  status               TEXT NOT NULL,     -- running, completed, failed
  started_at           BIGINT NOT NULL,   -- nanoseconds since epoch
  completed_at         BIGINT,            -- nanoseconds since epoch
  files_processed      INT NOT NULL DEFAULT 0,
  chunks_created       INT NOT NULL DEFAULT 0,
  embeddings_generated INT NOT NULL DEFAULT 0,
  embeddings_cached    INT NOT NULL DEFAULT 0,
  error_message        TEXT,
  duration_ms          BIGINT
);

CREATE INDEX IF NOT EXISTS idx_index_runs_repo_id ON index_runs(repo_id);
CREATE INDEX IF NOT EXISTS idx_index_runs_status ON index_runs(status);

CREATE TABLE IF NOT EXISTS settings (
  key       TEXT PRIMARY KEY,
  value     BYTEA NOT NULL,
  is_secret BOOLEAN NOT NULL DEFAULT false,
  updated   BIGINT NOT NULL -- nanoseconds since epoch
);

-- +goose Down
-- Drop in reverse dependency order.
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS index_runs;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS git_tokens;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
