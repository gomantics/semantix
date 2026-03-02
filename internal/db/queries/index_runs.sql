-- name: CreateIndexRun :one
INSERT INTO index_runs (repo_id, status, started_at)
VALUES ($1, $2, $3)
RETURNING id, repo_id, status, started_at, completed_at, files_processed, chunks_created, embeddings_generated, embeddings_cached, error_message, duration_ms;

-- name: GetIndexRunByID :one
SELECT id, repo_id, status, started_at, completed_at, files_processed, chunks_created, embeddings_generated, embeddings_cached, error_message, duration_ms
FROM index_runs
WHERE id = $1;

-- name: ListIndexRunsByRepo :many
SELECT id, repo_id, status, started_at, completed_at, files_processed, chunks_created, embeddings_generated, embeddings_cached, error_message, duration_ms
FROM index_runs
WHERE repo_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateIndexRunStatus :one
UPDATE index_runs
SET status = $2,
    completed_at = $3,
    error_message = $4,
    duration_ms = $5
WHERE id = $1
RETURNING id, repo_id, status, started_at, completed_at, files_processed, chunks_created, embeddings_generated, embeddings_cached, error_message, duration_ms;

-- name: UpdateIndexRunStats :one
UPDATE index_runs
SET files_processed = $2,
    chunks_created = $3,
    embeddings_generated = $4,
    embeddings_cached = $5
WHERE id = $1
RETURNING id, repo_id, status, started_at, completed_at, files_processed, chunks_created, embeddings_generated, embeddings_cached, error_message, duration_ms;

-- name: DeleteIndexRunsByRepo :exec
DELETE FROM index_runs
WHERE repo_id = $1;

-- name: FailOrphanedIndexRuns :exec
UPDATE index_runs
SET status = 'failed',
    completed_at = $1,
    error_message = 'interrupted by server restart'
WHERE status = 'running';
