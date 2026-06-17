-- +goose Up
CREATE TABLE shipping.shipments (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id    bigint NOT NULL,
    carrier     text NOT NULL,
    tracking_no text NOT NULL,
    status      text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE shipping.shipments;
