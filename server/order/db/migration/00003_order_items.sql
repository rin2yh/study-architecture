-- +goose Up
CREATE TABLE "order".order_items (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id         bigint NOT NULL REFERENCES "order".orders(id) ON DELETE CASCADE,
    product_id       bigint NOT NULL,
    -- [[0008]]
    product_name     text NOT NULL,
    unit_price_cents bigint NOT NULL,
    quantity         integer NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX order_items_order_id_idx ON "order".order_items (order_id);

-- +goose Down
DROP TABLE "order".order_items;
