-- name: ListProducts :many
SELECT id, sku, name, price_cents, created_at
FROM product.products
ORDER BY id;
