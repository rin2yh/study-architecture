-- +goose Up
CREATE TABLE product.products (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku         text NOT NULL UNIQUE,
    name        text NOT NULL,
    price_cents bigint NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE product.products;
