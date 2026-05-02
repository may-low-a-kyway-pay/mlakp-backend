-- name: CreateUser :one
INSERT INTO users (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id, name, email, password_hash, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, name, email, password_hash, created_at, updated_at
FROM users
WHERE id = $1;
