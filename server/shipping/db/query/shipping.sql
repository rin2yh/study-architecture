-- name: ListShipments :many
SELECT id, order_id, carrier, tracking_no, status, created_at
FROM shipping.shipments
ORDER BY id;

-- name: GetShipment :one
SELECT id, order_id, carrier, tracking_no, status, created_at
FROM shipping.shipments
WHERE id = $1;

-- name: CreateShipment :one
INSERT INTO shipping.shipments (order_id, carrier, tracking_no, status)
VALUES ($1, $2, $3, $4)
RETURNING id, order_id, carrier, tracking_no, status, created_at;

-- name: UpdateShipment :one
UPDATE shipping.shipments
SET status = $2
WHERE id = $1
RETURNING id, order_id, carrier, tracking_no, status, created_at;
