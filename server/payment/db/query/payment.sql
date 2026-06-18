-- name: ListPayments :many
SELECT id, order_id, amount_cents, method, status, created_at
FROM payment.payments
ORDER BY id;

-- name: GetPayment :one
SELECT id, order_id, amount_cents, method, status, created_at
FROM payment.payments
WHERE id = $1;

-- name: CreatePayment :one
INSERT INTO payment.payments (order_id, amount_cents, method, status)
VALUES ($1, $2, $3, $4)
RETURNING id, order_id, amount_cents, method, status, created_at;
