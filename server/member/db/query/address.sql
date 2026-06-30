-- name: ListAddresses :many
SELECT id, member_id, recipient, postal_code, prefecture, city, line1, created_at
FROM member.addresses
WHERE member_id = $1
ORDER BY id;

-- name: GetAddress :one
SELECT id, member_id, recipient, postal_code, prefecture, city, line1, created_at
FROM member.addresses
WHERE id = $1 AND member_id = $2;

-- name: CreateAddress :one
INSERT INTO member.addresses (member_id, recipient, postal_code, prefecture, city, line1)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, member_id, recipient, postal_code, prefecture, city, line1, created_at;

-- name: UpdateAddress :one
UPDATE member.addresses
SET recipient = $3, postal_code = $4, prefecture = $5, city = $6, line1 = $7
WHERE id = $1 AND member_id = $2
RETURNING id, member_id, recipient, postal_code, prefecture, city, line1, created_at;

-- name: DeleteAddress :exec
DELETE FROM member.addresses
WHERE id = $1 AND member_id = $2;
