-- name: CreateGroup :one
INSERT INTO groups (name, created_by)
VALUES ($1, $2)
RETURNING id, name, created_by, created_at, updated_at;

-- name: CreateGroupMember :one
INSERT INTO group_members (group_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING id, group_id, user_id, role, joined_at;

-- name: ListGroupsForUser :many
SELECT g.id, g.name, g.created_by, g.created_at, g.updated_at
FROM groups g
JOIN group_members gm ON gm.group_id = g.id
WHERE gm.user_id = $1
ORDER BY g.created_at DESC, g.id DESC;

-- name: GetGroupForUser :one
SELECT g.id, g.name, g.created_by, g.created_at, g.updated_at
FROM groups g
JOIN group_members gm ON gm.group_id = g.id
WHERE g.id = $1
  AND gm.user_id = $2;

-- name: ListGroupMembersForUser :many
SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.joined_at, u.name AS user_name, u.email AS user_email
FROM group_members gm
JOIN users u ON u.id = gm.user_id
JOIN group_members viewer ON viewer.group_id = gm.group_id
WHERE gm.group_id = $1
  AND viewer.user_id = $2
ORDER BY gm.joined_at ASC, gm.id ASC;

-- name: IsGroupOwner :one
SELECT EXISTS (
    SELECT 1
    FROM group_members
    WHERE group_id = $1
      AND user_id = $2
      AND role = 'owner'
);

-- name: UserExists :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE id = $1
);
