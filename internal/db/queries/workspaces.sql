-- name: CreateWorkspace :one
INSERT INTO workspaces (name, slug, description, settings, created, updated)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, slug, description, settings, created, updated;

-- name: GetWorkspaceByID :one
SELECT id, name, slug, description, settings, created, updated
FROM workspaces
WHERE id = $1;

-- name: GetWorkspaceBySlug :one
SELECT id, name, slug, description, settings, created, updated
FROM workspaces
WHERE slug = $1;

-- name: ListWorkspaces :many
SELECT id, name, slug, description, settings, created, updated
FROM workspaces
ORDER BY created DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkspaces :one
SELECT COUNT(*)
FROM workspaces;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = $2,
    slug = $3,
    description = $4,
    settings = $5,
    updated = $6
WHERE id = $1
RETURNING id, name, slug, description, settings, created, updated;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;
