-- name: CreateFile :one
INSERT INTO files (
    repo_id,
    path,
    shasum,
    language,
    size_bytes,
    chunk_count,
    indexed_at,
    created,
    updated
  )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;
-- name: GetFileByID :one
SELECT *
FROM files
WHERE id = $1;
-- name: GetFileByRepoAndPath :one
SELECT *
FROM files
WHERE repo_id = $1
  AND path = $2;
-- name: ListFilesByRepoID :many
SELECT *
FROM files
WHERE repo_id = $1
ORDER BY path ASC;
-- name: CountFilesByRepoID :one
SELECT COUNT(*)
FROM files
WHERE repo_id = $1;
-- name: SumChunksByRepoID :one
SELECT COALESCE(SUM(chunk_count), 0)::BIGINT AS total_chunks
FROM files
WHERE repo_id = $1;
-- name: UpdateFileShasum :one
UPDATE files
SET shasum = $2,
  chunk_count = $3,
  indexed_at = $4,
  updated = $5
WHERE id = $1
RETURNING *;
-- name: UpsertFile :one
INSERT INTO files (
    repo_id,
    path,
    shasum,
    language,
    size_bytes,
    chunk_count,
    indexed_at,
    created,
    updated
  )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (repo_id, path) DO
UPDATE
SET shasum = EXCLUDED.shasum,
  language = EXCLUDED.language,
  size_bytes = EXCLUDED.size_bytes,
  chunk_count = EXCLUDED.chunk_count,
  indexed_at = EXCLUDED.indexed_at,
  updated = EXCLUDED.updated
RETURNING *;
-- name: DeleteFile :exec
DELETE FROM files
WHERE id = $1;
-- name: DeleteFilesByRepoID :exec
DELETE FROM files
WHERE repo_id = $1;
-- name: GetFilesWithDifferentShasum :many
SELECT f.*
FROM files f
WHERE f.repo_id = $1
  AND f.shasum != ALL($2::TEXT []);