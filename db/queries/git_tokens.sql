-- name: CreateGitToken :one
INSERT INTO git_tokens (
    workspace_id,
    provider,
    name,
    token_encrypted,
    created,
    updated
  )
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
-- name: GetGitTokenByID :one
SELECT *
FROM git_tokens
WHERE id = $1;
-- name: ListGitTokensByWorkspace :many
SELECT *
FROM git_tokens
WHERE workspace_id = $1
ORDER BY created DESC;
-- name: ListGitTokensByWorkspaceAndProvider :many
SELECT *
FROM git_tokens
WHERE workspace_id = $1
  AND provider = $2
ORDER BY created DESC;
-- name: UpdateGitToken :one
UPDATE git_tokens
SET name = $2,
  token_encrypted = $3,
  updated = $4
WHERE id = $1
RETURNING *;
-- name: DeleteGitToken :exec
DELETE FROM git_tokens
WHERE id = $1;
-- name: DeleteGitTokensByWorkspace :exec
DELETE FROM git_tokens
WHERE workspace_id = $1;