-- name: CreateGitToken :one
INSERT INTO git_tokens (name, provider, token_encrypted, token_hint, created)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, provider, token_encrypted, token_hint, created;

-- name: GetGitTokenByID :one
SELECT id, name, provider, token_encrypted, token_hint, created
FROM git_tokens
WHERE id = $1;

-- name: ListGitTokens :many
SELECT id, name, provider, token_encrypted, token_hint, created
FROM git_tokens
ORDER BY created DESC;

-- name: ListGitTokensByProvider :many
SELECT id, name, provider, token_encrypted, token_hint, created
FROM git_tokens
WHERE provider = $1
ORDER BY created DESC;

-- name: DeleteGitToken :exec
DELETE FROM git_tokens
WHERE id = $1;
