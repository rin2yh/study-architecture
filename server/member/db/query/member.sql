-- name: ListMembers :many
SELECT id, email, display_name, created_at, password_hash
FROM member.members
ORDER BY id;

-- name: GetMember :one
SELECT id, email, display_name, created_at, password_hash
FROM member.members
WHERE id = $1;

-- name: GetMemberByEmail :one
SELECT id, email, display_name, created_at, password_hash
FROM member.members
WHERE email = $1;

-- name: CreateMember :one
INSERT INTO member.members (email, display_name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, display_name, created_at, password_hash;

-- name: UpdateMember :one
UPDATE member.members
SET email = $2, display_name = $3
WHERE id = $1
RETURNING id, email, display_name, created_at, password_hash;
