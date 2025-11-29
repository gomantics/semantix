-- name: CreateWorkspace :one
INSERT INTO workspaces (name, slug, created, updated)
VALUES ($1, $2, $3, $4)
RETURNING *;
-- name: GetWorkspaceByID :one
SELECT *
FROM workspaces
WHERE id = $1;
-- name: GetWorkspaceBySlug :one
SELECT *
FROM workspaces
WHERE slug = $1;
-- name: ListWorkspaces :many
SELECT *
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
  updated = $4
WHERE id = $1
RETURNING *;
-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;