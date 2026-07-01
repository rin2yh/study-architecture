-- +goose Up
-- ADR-[[202606261704]]
ALTER TABLE shipping.shipments
    ADD COLUMN ship_recipient   text NOT NULL DEFAULT '',
    ADD COLUMN ship_postal_code text NOT NULL DEFAULT '',
    ADD COLUMN ship_prefecture  text NOT NULL DEFAULT '',
    ADD COLUMN ship_city        text NOT NULL DEFAULT '',
    ADD COLUMN ship_line1       text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE shipping.shipments
    DROP COLUMN ship_recipient,
    DROP COLUMN ship_postal_code,
    DROP COLUMN ship_prefecture,
    DROP COLUMN ship_city,
    DROP COLUMN ship_line1;
