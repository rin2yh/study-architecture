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
INSERT INTO payment.payments (
    order_id, amount_cents, method, status, idempotency_key,
    settled_event_ship_recipient, settled_event_ship_postal_code,
    settled_event_ship_prefecture, settled_event_ship_city, settled_event_ship_line1
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (idempotency_key) WHERE idempotency_key <> '' DO NOTHING
RETURNING *;

-- name: UpdatePayment :one
UPDATE payment.payments
SET status                    = sqlc.arg(status),
    settled_event_pending     = settled_event_pending OR sqlc.arg(mark_settled),
    settled_event_traceparent = CASE WHEN sqlc.arg(mark_settled) THEN sqlc.arg(traceparent) ELSE settled_event_traceparent END
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListUnpublishedSettledEvents :many
SELECT id, order_id, amount_cents, settled_event_traceparent,
       settled_event_ship_recipient, settled_event_ship_postal_code,
       settled_event_ship_prefecture, settled_event_ship_city, settled_event_ship_line1
FROM payment.payments
WHERE settled_event_pending
ORDER BY id
LIMIT $1;

-- name: MarkSettledEventPublished :exec
UPDATE payment.payments
SET settled_event_pending = false, settled_event_published_at = now()
WHERE id = $1;
