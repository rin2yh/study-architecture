-- name: ListMembers :many
SELECT id, email, display_name, created_at
FROM member.members
ORDER BY id;
