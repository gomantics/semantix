-- Migration: 00001_initial_schema
-- Safe to re-run: yes (all statements use IF NOT EXISTS)

-- +goose Up
CREATE TABLE IF NOT EXISTS workspaces (
  id          BIGSERIAL PRIMARY KEY,
  name        TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  description TEXT,
  settings    JSONB NOT NULL,
  created     BIGINT NOT NULL,  -- nanoseconds since epoch
  updated     BIGINT NOT NULL   -- nanoseconds since epoch
);

CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);

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
  url           TEXT NOT NULL,
  branch        TEXT NOT NULL,
  status        TEXT NOT NULL,  -- pending, indexing, ready, error
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

-- +goose Down
-- Drop in reverse dependency order (repos/files reference workspaces/git_tokens).
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS git_tokens;
DROP TABLE IF EXISTS workspaces;
