-- name: ListMembers :many
SELECT id, email, display_name, created_at
FROM member.members
ORDER BY id;

-- name: GetMember :one
SELECT id, email, display_name, created_at
FROM member.members
WHERE id = $1;

-- name: CreateMember :one
INSERT INTO member.members (email, display_name)
VALUES ($1, $2)
RETURNING id, email, display_name, created_at;
