-- name: ListPayments :many
SELECT * FROM payment.payments
ORDER BY id;

-- name: GetPayment :one
SELECT * FROM payment.payments
WHERE id = $1;

-- name: GetPaymentByIdempotencyKey :one
SELECT * FROM payment.payments
WHERE idempotency_key = $1;

-- ADR-[[202606261214]]
-- name: CreatePayment :one
INSERT INTO payment.payments (order_id, amount_cents, method, status, idempotency_key)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (idempotency_key) WHERE idempotency_key <> '' DO NOTHING
RETURNING *;

-- name: UpdatePayment :one
UPDATE payment.payments
SET status = $2
WHERE id = $1
RETURNING *;

-- name: InsertOutbox :exec
INSERT INTO payment.outbox (aggregate_id, event_type, payload, traceparent)
VALUES ($1, $2, $3, $4);

-- name: ListUnpublishedOutbox :many
SELECT id, payload, traceparent
FROM payment.outbox
WHERE published_at IS NULL
ORDER BY id
LIMIT $1;

-- name: MarkOutboxPublished :exec
UPDATE payment.outbox
SET published_at = now()
WHERE id = $1;

-- 終端でない確定済み決済だけ返金へ遷移。再配信は 0 行更新で吸収する (ADR-[[202606261214]])。
-- name: RefundPaymentByOrder :exec
UPDATE payment.payments
SET status = 'refunded'
WHERE order_id = $1 AND status IN ('paid', 'settled', 'captured');

-- 返金すべき入金がないため refunded とは区別する (ADR-[[202606261702]])。
-- name: VoidPendingPaymentByOrder :exec
UPDATE payment.payments
SET status = 'cancelled'
WHERE order_id = $1 AND status = 'pending';
