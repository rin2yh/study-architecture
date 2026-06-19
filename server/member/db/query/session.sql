-- name: CreateSession :one
INSERT INTO member.sessions (id, member_id, expires_at)
VALUES ($1, $2, $3)
RETURNING id, member_id, expires_at, created_at;

-- name: GetSession :one
SELECT id, member_id, expires_at, created_at
FROM member.sessions
WHERE id = $1 AND expires_at > now();

-- name: DeleteSession :exec
DELETE FROM member.sessions
WHERE id = $1;
