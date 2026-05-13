-- name: CreateAuthSession :one
INSERT INTO auth_sessions (user_id, refresh_token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, refresh_token_hash, created_at, expires_at, revoked_at, last_used_at;

-- name: GetActiveAuthSession :one
SELECT id, user_id, refresh_token_hash, created_at, expires_at, revoked_at, last_used_at
FROM auth_sessions
WHERE id = $1
  AND revoked_at IS NULL
  AND expires_at > now();

-- name: RotateRefreshToken :one
UPDATE auth_sessions
SET refresh_token_hash = $2,
    last_used_at = now()
WHERE refresh_token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > now()
RETURNING id, user_id, refresh_token_hash, created_at, expires_at, revoked_at, last_used_at;

-- name: RevokeAuthSession :exec
UPDATE auth_sessions
SET revoked_at = now()
WHERE id = $1
  AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE auth_sessions
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;
