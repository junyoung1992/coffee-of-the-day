-- name: InsertLog :exec
INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, memo, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetLogByID :one
SELECT id, user_id, recorded_at, companions, log_type, memo, created_at, updated_at
FROM coffee_logs
WHERE id = ? AND user_id = ?;

-- name: ListLogs :many
SELECT id, user_id, recorded_at, companions, log_type, memo, created_at, updated_at
FROM coffee_logs
WHERE user_id = ?
  AND (sqlc.narg('log_type') IS NULL OR log_type = sqlc.narg('log_type'))
  AND (sqlc.narg('date_from') IS NULL OR recorded_at >= sqlc.narg('date_from'))
  AND (sqlc.narg('date_to') IS NULL OR recorded_at <= sqlc.narg('date_to'))
  AND (
    sqlc.narg('cursor_recorded_at') IS NULL
    OR recorded_at < sqlc.narg('cursor_recorded_at')
    OR (recorded_at = sqlc.narg('cursor_recorded_at') AND id < sqlc.narg('cursor_id'))
  )
ORDER BY recorded_at DESC, id DESC
LIMIT ?;

-- name: UpdateLog :exec
UPDATE coffee_logs
SET recorded_at = ?, companions = ?, memo = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteLog :exec
DELETE FROM coffee_logs
WHERE id = ? AND user_id = ?;
