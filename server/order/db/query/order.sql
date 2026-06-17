-- name: ListOrders :many
SELECT id, member_id, status, total_cents, created_at
FROM "order".orders
ORDER BY id;
