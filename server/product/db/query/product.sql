-- name: ListProducts :many
SELECT id, sku, name, price_cents, created_at
FROM product.products
ORDER BY id;

-- name: GetProduct :one
SELECT id, sku, name, price_cents, created_at
FROM product.products
WHERE id = $1;

-- name: CreateProduct :one
INSERT INTO product.products (sku, name, price_cents)
VALUES ($1, $2, $3)
RETURNING id, sku, name, price_cents, created_at;

-- name: UpdateProduct :one
UPDATE product.products
SET sku = $2, name = $3, price_cents = $4
WHERE id = $1
RETURNING id, sku, name, price_cents, created_at;
