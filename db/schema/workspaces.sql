CREATE TABLE IF NOT EXISTS workspaces (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  created BIGINT NOT NULL,
  updated BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_workspaces_slug ON workspaces(slug);