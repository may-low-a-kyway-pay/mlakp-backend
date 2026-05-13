-- name: CreateUser :one
INSERT INTO users (name, username, email, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, name, username, email, password_hash, email_verified_at, verification_deadline, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, name, username, email, password_hash, email_verified_at, verification_deadline, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, name, username, email, password_hash, email_verified_at, verification_deadline, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, name, username, email, password_hash, email_verified_at, verification_deadline, created_at, updated_at
FROM users
WHERE username = $1;

-- name: SearchUsersByUsername :many
SELECT id, name, username, email, password_hash, email_verified_at, verification_deadline, created_at, updated_at
FROM users
-- Treat underscores as literal username characters instead of SQL LIKE wildcards.
WHERE username LIKE replace($1::text, '_', '\_') || '%' ESCAPE '\'
ORDER BY username ASC
LIMIT $2;

-- name: UpdateUserUsername :one
UPDATE users
SET username = $2
WHERE id = $1
RETURNING id, name, username, email, password_hash, email_verified_at, verification_deadline, created_at, updated_at;

-- name: MarkEmailVerified :one
UPDATE users
SET email_verified_at = now()
WHERE id = $1
RETURNING id, name, username, email, email_verified_at, verification_deadline, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;
