-- name: ListPayments :many
SELECT id, order_id, amount_cents, method, status, created_at
FROM payment.payments
ORDER BY id;
