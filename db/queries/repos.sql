-- name: CreateRepo :one
INSERT INTO repos (
    workspace_id,
    git_token_id,
    url,
    name,
    owner,
    default_branch,
    status,
    created,
    updated
  )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;
-- name: GetRepoByID :one
SELECT *
FROM repos
WHERE id = $1;
-- name: GetRepoByWorkspaceAndURL :one
SELECT *
FROM repos
WHERE workspace_id = $1
  AND url = $2;
-- name: ListReposByWorkspace :many
SELECT *
FROM repos
WHERE workspace_id = $1
ORDER BY created DESC
LIMIT $2 OFFSET $3;
-- name: ListReposByWorkspaceAndStatus :many
SELECT *
FROM repos
WHERE workspace_id = $1
  AND status = $2
ORDER BY created DESC
LIMIT $3 OFFSET $4;
-- name: CountReposByWorkspace :one
SELECT COUNT(*)
FROM repos
WHERE workspace_id = $1;
-- name: CountReposByWorkspaceAndStatus :one
SELECT COUNT(*)
FROM repos
WHERE workspace_id = $1
  AND status = $2;
-- name: UpdateRepoStatus :one
UPDATE repos
SET status = $2,
  error = $3,
  updated = $4
WHERE id = $1
RETURNING *;
-- name: UpdateRepoAfterClone :one
UPDATE repos
SET last_commit_sha = $2,
  status = $3,
  updated = $4
WHERE id = $1
RETURNING *;
-- name: UpdateRepoGitToken :one
UPDATE repos
SET git_token_id = $2,
  updated = $3
WHERE id = $1
RETURNING *;
-- name: DeleteRepo :exec
DELETE FROM repos
WHERE id = $1;
-- name: DeleteReposByWorkspace :exec
DELETE FROM repos
WHERE workspace_id = $1;
-- name: GetPendingRepos :many
SELECT *
FROM repos
WHERE status = 'pending'
ORDER BY created ASC
LIMIT $1;
-- name: ClaimPendingRepo :one
UPDATE repos
SET status = 'indexing',
  updated = $1
WHERE id = (
    SELECT id
    FROM repos
    WHERE status = 'pending'
    ORDER BY created ASC
    LIMIT 1 FOR
    UPDATE SKIP LOCKED
  )
RETURNING *;