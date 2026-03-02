-- name: CreateFile :one
INSERT INTO files (repo_id, path, content_hash, size_bytes, language, indexed_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, repo_id, path, content_hash, size_bytes, language, indexed_at;

-- name: GetFileByID :one
SELECT id, repo_id, path, content_hash, size_bytes, language, indexed_at
FROM files
WHERE id = $1;

-- name: GetFileByRepoAndPath :one
SELECT id, repo_id, path, content_hash, size_bytes, language, indexed_at
FROM files
WHERE repo_id = $1 AND path = $2;

-- name: ListFilesByRepo :many
SELECT id, repo_id, path, content_hash, size_bytes, language, indexed_at
FROM files
WHERE repo_id = $1
ORDER BY path ASC;

-- name: CountFilesByRepo :one
SELECT COUNT(*)
FROM files
WHERE repo_id = $1;

-- name: UpsertFile :one
INSERT INTO files (repo_id, path, content_hash, size_bytes, language, indexed_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (repo_id, path) DO UPDATE SET
    content_hash = EXCLUDED.content_hash,
    size_bytes = EXCLUDED.size_bytes,
    language = EXCLUDED.language,
    indexed_at = EXCLUDED.indexed_at
RETURNING id, repo_id, path, content_hash, size_bytes, language, indexed_at;

-- name: DeleteFile :exec
DELETE FROM files
WHERE id = $1;

-- name: DeleteFilesByRepo :exec
DELETE FROM files
WHERE repo_id = $1;

-- name: DeleteFilesByPaths :exec
DELETE FROM files
WHERE repo_id = $1 AND path = ANY($2::text[]);

-- name: DeleteStaleFiles :exec
DELETE FROM files
WHERE repo_id = $1 AND id != ALL($2::bigint[]);
