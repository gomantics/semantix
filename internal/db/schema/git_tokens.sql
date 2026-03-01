CREATE TABLE IF NOT EXISTS git_tokens (
  id              BIGSERIAL PRIMARY KEY,
  name            TEXT NOT NULL,
  provider        TEXT NOT NULL, -- github, gitlab, bitbucket
  token_encrypted BYTEA NOT NULL,
  created         BIGINT NOT NULL
);
