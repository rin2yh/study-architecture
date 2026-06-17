-- +goose Up
CREATE TABLE "order".orders (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    member_id   bigint NOT NULL,
    status      text NOT NULL,
    total_cents bigint NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE "order".orders;
