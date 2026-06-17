-- +goose Up
CREATE TABLE payment.payments (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id     bigint NOT NULL,
    amount_cents bigint NOT NULL,
    method       text NOT NULL,
    status       text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE payment.payments;
