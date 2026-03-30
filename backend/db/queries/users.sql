-- name: CreateUser :one
INSERT INTO users (id, username, display_name, email, password_hash, created_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: IncrementTokenVersion :exec
UPDATE users SET token_version = token_version + 1 WHERE id = ?;
