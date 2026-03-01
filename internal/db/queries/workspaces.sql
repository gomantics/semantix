-- name: CreateWorkspace :one
INSERT INTO workspaces (name, description, settings, created, updated)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, description, settings, created, updated;

-- name: GetWorkspaceByID :one
SELECT id, name, description, settings, created, updated
FROM workspaces
WHERE id = $1;

-- name: ListWorkspaces :many
SELECT id, name, description, settings, created, updated
FROM workspaces
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkspaces :one
SELECT COUNT(*)
FROM workspaces;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = $2,
    description = $3,
    settings = $4,
    updated = $5
WHERE id = $1
RETURNING id, name, description, settings, created, updated;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;
