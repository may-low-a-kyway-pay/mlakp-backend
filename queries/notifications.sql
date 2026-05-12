-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, title, body, entity_type, entity_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, user_id, type, title, body, entity_type, entity_id, metadata, read_at, created_at;

-- name: ListNotificationsForUser :many
SELECT id, user_id, type, title, body, entity_type, entity_id, metadata, read_at, created_at
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: CountUnreadNotificationsForUser :one
SELECT count(*)::bigint
FROM notifications
WHERE user_id = $1
  AND read_at IS NULL;

-- name: MarkNotificationRead :one
UPDATE notifications
SET read_at = COALESCE(read_at, now())
WHERE id = $1
  AND user_id = $2
RETURNING id, user_id, type, title, body, entity_type, entity_id, metadata, read_at, created_at;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET read_at = COALESCE(read_at, now())
WHERE user_id = $1
  AND read_at IS NULL;
