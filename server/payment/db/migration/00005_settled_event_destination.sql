-- +goose Up
-- 配送先スナップショットの中継列 (ADR-[[202606261704]])。payment は解釈せず traceparent と同様に
-- settled イベントへ載せて中継するだけ。
ALTER TABLE payment.payments
    ADD COLUMN settled_event_ship_recipient   text NOT NULL DEFAULT '',
    ADD COLUMN settled_event_ship_postal_code text NOT NULL DEFAULT '',
    ADD COLUMN settled_event_ship_prefecture  text NOT NULL DEFAULT '',
    ADD COLUMN settled_event_ship_city        text NOT NULL DEFAULT '',
    ADD COLUMN settled_event_ship_line1       text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE payment.payments
    DROP COLUMN settled_event_ship_recipient,
    DROP COLUMN settled_event_ship_postal_code,
    DROP COLUMN settled_event_ship_prefecture,
    DROP COLUMN settled_event_ship_city,
    DROP COLUMN settled_event_ship_line1;
