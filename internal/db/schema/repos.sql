CREATE TABLE IF NOT EXISTS repos (
  id            BIGSERIAL PRIMARY KEY,
  workspace_id  BIGINT NOT NULL,
  url           TEXT NOT NULL,
  branch        TEXT NOT NULL,
  status        TEXT NOT NULL,  -- pending, indexing, ready, error
  indexed_at    BIGINT,
  error_message TEXT,
  created       BIGINT NOT NULL,
  updated       BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_repos_workspace_id ON repos(workspace_id);
CREATE INDEX IF NOT EXISTS idx_repos_status ON repos(status);
