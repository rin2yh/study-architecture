-- name: ListOrders :many
SELECT id, member_id, status, total_cents, created_at
FROM "order".orders
ORDER BY id;

-- name: GetOrder :one
SELECT id, member_id, status, total_cents, created_at
FROM "order".orders
WHERE id = $1;

-- name: CreateOrder :one
INSERT INTO "order".orders (member_id, status, total_cents)
VALUES ($1, $2, $3)
RETURNING id, member_id, status, total_cents, created_at;

-- name: UpdateOrder :one
UPDATE "order".orders
SET status = $2, total_cents = $3
WHERE id = $1
RETURNING id, member_id, status, total_cents, created_at;
