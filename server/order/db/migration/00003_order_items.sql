-- +goose Up
CREATE TABLE "order".order_items (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id         bigint NOT NULL REFERENCES "order".orders(id) ON DELETE CASCADE,
    product_id       bigint NOT NULL,
    -- product_name / unit_price_cents は注文確定時に product から複写したスナップショット。
    -- product 側の改名・改価は確定済みの注文に遡及させない ([[0008]])。
    product_name     text NOT NULL,
    unit_price_cents bigint NOT NULL,
    quantity         integer NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX order_items_order_id_idx ON "order".order_items (order_id);

-- +goose Down
DROP TABLE "order".order_items;
