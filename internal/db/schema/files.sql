CREATE TABLE IF NOT EXISTS files (
  id           BIGSERIAL PRIMARY KEY,
  repo_id      BIGINT NOT NULL,
  path         TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  size_bytes   BIGINT NOT NULL,
  language     TEXT,
  indexed_at   BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_files_repo_id ON files(repo_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_repo_path ON files(repo_id, path);
