CREATE TABLE IF NOT EXISTS repos (
  id BIGSERIAL PRIMARY KEY,
  workspace_id BIGINT NOT NULL,
  git_token_id BIGINT,
  url TEXT NOT NULL,
  name TEXT NOT NULL,
  owner TEXT NOT NULL,
  default_branch TEXT NOT NULL,
  last_commit_sha TEXT,
  status TEXT NOT NULL,
  error TEXT,
  created BIGINT NOT NULL,
  updated BIGINT NOT NULL,
  UNIQUE(workspace_id, url)
);
CREATE INDEX IF NOT EXISTS idx_repos_workspace_id ON repos(workspace_id);
CREATE INDEX IF NOT EXISTS idx_repos_status ON repos(status);