CREATE TABLE IF NOT EXISTS git_tokens (
  id BIGSERIAL PRIMARY KEY,
  workspace_id BIGINT NOT NULL,
  provider TEXT NOT NULL,
  name TEXT NOT NULL,
  token_encrypted TEXT NOT NULL,
  created BIGINT NOT NULL,
  updated BIGINT NOT NULL,
  UNIQUE(workspace_id, provider, name)
);
CREATE INDEX IF NOT EXISTS idx_git_tokens_workspace_id ON git_tokens(workspace_id);