-- name: CreateUser :one
INSERT INTO users (email, password_hash, created, updated)
VALUES ($1, $2, $3, $4)
RETURNING id, email, password_hash, created, updated;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, created, updated
FROM users
WHERE email = $1;

-- name: CountUsers :one
SELECT COUNT(*)
FROM users;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token, created, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, token, created, expires_at;

-- name: GetSessionByToken :one
SELECT s.id, s.user_id, s.token, s.created, s.expires_at,
       u.email
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token = $1
  AND (s.expires_at IS NULL OR s.expires_at > sqlc.arg(now));

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE token = $1;
