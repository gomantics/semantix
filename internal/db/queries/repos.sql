-- name: CreateRepo :one
INSERT INTO repos (workspace_id, url, branch, status, indexed_at, error_message, created, updated)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, workspace_id, url, branch, status, indexed_at, error_message, created, updated;

-- name: GetRepoByID :one
SELECT id, workspace_id, url, branch, status, indexed_at, error_message, created, updated
FROM repos
WHERE id = $1;

-- name: ListReposByWorkspace :many
SELECT id, workspace_id, url, branch, status, indexed_at, error_message, created, updated
FROM repos
WHERE workspace_id = $1
ORDER BY created DESC
LIMIT $2 OFFSET $3;

-- name: ListReposByStatus :many
SELECT id, workspace_id, url, branch, status, indexed_at, error_message, created, updated
FROM repos
WHERE status = $1
ORDER BY created ASC
LIMIT $2;

-- name: CountReposByWorkspace :one
SELECT COUNT(*)
FROM repos
WHERE workspace_id = $1;

-- name: UpdateRepoStatus :one
UPDATE repos
SET status = $2,
    indexed_at = $3,
    error_message = $4,
    updated = $5
WHERE id = $1
RETURNING id, workspace_id, url, branch, status, indexed_at, error_message, created, updated;

-- name: UpdateRepo :one
UPDATE repos
SET url = $2,
    branch = $3,
    updated = $4
WHERE id = $1
RETURNING id, workspace_id, url, branch, status, indexed_at, error_message, created, updated;

-- name: DeleteRepo :exec
DELETE FROM repos
WHERE id = $1;

-- name: DeleteReposByWorkspace :exec
DELETE FROM repos
WHERE workspace_id = $1;
