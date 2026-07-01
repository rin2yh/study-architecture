-- +goose Up
-- 注文時点の配送先スナップショット (ADR-[[202606261704]] / ADR-[[202606190900]])。
ALTER TABLE "order".orders
    ADD COLUMN shipping_recipient   text NOT NULL DEFAULT '',
    ADD COLUMN shipping_postal_code text NOT NULL DEFAULT '',
    ADD COLUMN shipping_prefecture  text NOT NULL DEFAULT '',
    ADD COLUMN shipping_city        text NOT NULL DEFAULT '',
    ADD COLUMN shipping_line1       text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE "order".orders
    DROP COLUMN shipping_recipient,
    DROP COLUMN shipping_postal_code,
    DROP COLUMN shipping_prefecture,
    DROP COLUMN shipping_city,
    DROP COLUMN shipping_line1;
