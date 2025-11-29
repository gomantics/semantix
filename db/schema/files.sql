CREATE TABLE IF NOT EXISTS files (
  id BIGSERIAL PRIMARY KEY,
  repo_id BIGINT NOT NULL,
  path TEXT NOT NULL,
  shasum TEXT NOT NULL,
  language TEXT,
  size_bytes BIGINT NOT NULL,
  chunk_count INT NOT NULL,
  indexed_at BIGINT,
  created BIGINT NOT NULL,
  updated BIGINT NOT NULL,
  UNIQUE(repo_id, path)
);
CREATE INDEX IF NOT EXISTS idx_files_repo_id ON files(repo_id);
CREATE INDEX IF NOT EXISTS idx_files_shasum ON files(shasum);