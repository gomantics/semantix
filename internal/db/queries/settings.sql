-- name: UpsertSetting :one
INSERT INTO settings (key, value, is_secret, updated)
VALUES ($1, $2, $3, $4)
ON CONFLICT (key) DO UPDATE
  SET value     = EXCLUDED.value,
      is_secret = EXCLUDED.is_secret,
      updated   = EXCLUDED.updated
RETURNING key, value, is_secret, updated;

-- name: GetSettingByKey :one
SELECT key, value, is_secret, updated
FROM settings
WHERE key = $1;

-- name: ListSettings :many
SELECT key, value, is_secret, updated
FROM settings
ORDER BY key ASC;

-- name: DeleteSetting :exec
DELETE FROM settings
WHERE key = $1;
