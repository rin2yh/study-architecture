-- name: ListShipments :many
SELECT id, order_id, carrier, tracking_no, status, created_at
FROM shipping.shipments
ORDER BY id;
