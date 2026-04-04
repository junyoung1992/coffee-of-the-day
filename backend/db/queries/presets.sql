-- name: InsertPreset :exec
INSERT INTO presets (id, user_id, name, log_type, last_used_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetPresetByID :one
SELECT id, user_id, name, log_type, last_used_at, created_at, updated_at
FROM presets
WHERE id = ? AND user_id = ?;

-- name: ListPresetsByUserID :many
SELECT id, user_id, name, log_type, last_used_at, created_at, updated_at
FROM presets
WHERE user_id = ?
ORDER BY CASE WHEN last_used_at IS NULL THEN 1 ELSE 0 END, last_used_at DESC, created_at DESC;

