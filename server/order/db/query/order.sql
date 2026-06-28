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

-- 判定〜更新を 1 tx で直列化する行ロック付き取得 (ADR-[[202606261702]])。
-- name: GetOrderForUpdate :one
SELECT * FROM "order".orders
WHERE id = $1
FOR UPDATE;

-- 遷移と送信状態を同一 tx で確定し、送出はリレーに後追いさせる (ADR-[[202606261212]])。
-- name: CancelOrder :one
UPDATE "order".orders
SET status                      = 'cancelled',
    cancelled_event_pending     = true,
    cancelled_event_traceparent = $2
WHERE id = $1
RETURNING *;

-- name: ListUnpublishedCancelledEvents :many
SELECT id, cancelled_event_traceparent
FROM "order".orders
WHERE cancelled_event_pending
ORDER BY id
LIMIT $1;

-- name: MarkCancelledEventPublished :exec
UPDATE "order".orders
SET cancelled_event_pending = false, cancelled_event_published_at = now()
WHERE id = $1;
