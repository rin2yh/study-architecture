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
SET status = $2
WHERE id = $1
RETURNING id, member_id, status, total_cents, created_at;

-- name: CreateOrderItem :one
INSERT INTO "order".order_items (order_id, product_id, product_name, unit_price_cents, quantity)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, order_id, product_id, product_name, unit_price_cents, quantity, created_at;

-- name: ListOrderItems :many
SELECT id, order_id, product_id, product_name, unit_price_cents, quantity, created_at
FROM "order".order_items
WHERE order_id = $1
ORDER BY id;
