-- name: ListShipments :many
SELECT id, order_id, carrier, tracking_no, status, created_at, ship_recipient, ship_postal_code, ship_prefecture, ship_city, ship_line1
FROM shipping.shipments
ORDER BY id;

-- name: GetShipment :one
SELECT id, order_id, carrier, tracking_no, status, created_at, ship_recipient, ship_postal_code, ship_prefecture, ship_city, ship_line1
FROM shipping.shipments
WHERE id = $1;

-- name: CreateShipment :one
INSERT INTO shipping.shipments (order_id, carrier, tracking_no, status)
VALUES ($1, $2, $3, $4)
RETURNING id, order_id, carrier, tracking_no, status, created_at, ship_recipient, ship_postal_code, ship_prefecture, ship_city, ship_line1;

-- ADR-[[202606211200]] / ADR-[[202606261704]]
-- DO NOTHING は再配信の吸収に加え、先着した cancelled 行への手配も弾く (ADR-[[202608160810]])。
-- name: CreateShipmentForOrder :one
INSERT INTO shipping.shipments (order_id, ship_recipient, ship_postal_code, ship_prefecture, ship_city, ship_line1)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (order_id) DO NOTHING
RETURNING id, order_id, carrier, tracking_no, status, created_at, ship_recipient, ship_postal_code, ship_prefecture, ship_city, ship_line1;

-- name: UpdateShipment :one
UPDATE shipping.shipments
SET status = $2
WHERE id = $1
RETURNING id, order_id, carrier, tracking_no, status, created_at, ship_recipient, ship_postal_code, ship_prefecture, ship_city, ship_line1;

-- (ADR-[[202606261702]] / ADR-[[202606261214]] / ADR-[[202608160810]])
-- name: CancelShipmentForOrder :exec
INSERT INTO shipping.shipments (order_id, status)
VALUES ($1, 'cancelled')
ON CONFLICT (order_id) DO UPDATE
SET status = 'cancelled'
WHERE shipments.status = 'preparing';
