-- name: ListOrders :many
SELECT id, member_id, status, total_cents, created_at, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1
FROM "order".orders
ORDER BY id;

-- name: ListOrdersByMember :many
SELECT id, member_id, status, total_cents, created_at, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1
FROM "order".orders
WHERE member_id = $1
ORDER BY id;

-- name: GetOrder :one
SELECT id, member_id, status, total_cents, created_at, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1
FROM "order".orders
WHERE id = $1;

-- name: CreateOrder :one
INSERT INTO "order".orders (member_id, status, total_cents)
VALUES ($1, $2, $3)
RETURNING id, member_id, status, total_cents, created_at, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1;

-- name: CreateOrderWithShipping :one
INSERT INTO "order".orders (member_id, status, total_cents, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, member_id, status, total_cents, created_at, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1;

-- name: UpdateOrder :one
UPDATE "order".orders
SET status = $2
WHERE id = $1
RETURNING id, member_id, status, total_cents, created_at, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1;

-- name: CreateOrderItem :one
INSERT INTO "order".order_items (order_id, product_id, product_name, unit_price_cents, quantity)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, order_id, product_id, product_name, unit_price_cents, quantity, created_at;

-- name: ListOrderItems :many
SELECT id, order_id, product_id, product_name, unit_price_cents, quantity, created_at
FROM "order".order_items
WHERE order_id = $1
ORDER BY id;

-- name: DeleteOrder :exec
DELETE FROM "order".orders
WHERE id = $1;
