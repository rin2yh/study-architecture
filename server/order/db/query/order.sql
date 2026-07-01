-- name: ListOrders :many
SELECT * FROM "order".orders
ORDER BY id;

-- name: ListOrdersByMember :many
SELECT * FROM "order".orders
WHERE member_id = $1
ORDER BY id;

-- name: GetOrder :one
SELECT * FROM "order".orders
WHERE id = $1;

-- name: CreateOrder :one
INSERT INTO "order".orders (member_id, status, total_cents)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateOrderWithShipping :one
INSERT INTO "order".orders (member_id, status, total_cents, shipping_recipient, shipping_postal_code, shipping_prefecture, shipping_city, shipping_line1)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateOrder :one
UPDATE "order".orders
SET status = $2
WHERE id = $1
RETURNING *;

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

-- 判定〜更新を 1 tx で直列化する (ADR-[[202606261702]])。
-- name: GetOrderForUpdate :one
SELECT * FROM "order".orders
WHERE id = $1
FOR UPDATE;

-- 遷移と未送信イベントを同一 tx で確定し、送出はリレーに後追いさせる (ADR-[[202606300600]])。
-- name: CancelOrder :one
UPDATE "order".orders
SET status = 'cancelled'
WHERE id = $1
RETURNING *;

-- name: InsertOutbox :exec
INSERT INTO "order".outbox (aggregate_id, event_type, payload, traceparent)
VALUES ($1, $2, $3, $4);

-- name: ListUnpublishedOutbox :many
SELECT id, payload, traceparent
FROM "order".outbox
WHERE published_at IS NULL
ORDER BY id
LIMIT $1;

-- name: MarkOutboxPublished :exec
UPDATE "order".outbox
SET published_at = now()
WHERE id = $1;
