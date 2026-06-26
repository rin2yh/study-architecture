-- +goose Up
-- 決済確定イベントの Transactional Outbox 用の送信状態列 (ADR-[[202606261212]])。
ALTER TABLE payment.payments
    ADD COLUMN settled_event_pending      boolean     NOT NULL DEFAULT false,
    ADD COLUMN settled_event_traceparent  text        NOT NULL DEFAULT '',
    ADD COLUMN settled_event_published_at timestamptz;

-- 確定済みが大半を占めるため、リレーが走査する未送信行だけを対象にした部分インデックスにする。
CREATE INDEX payments_settled_event_pending_idx
    ON payment.payments (id)
    WHERE settled_event_pending;

-- +goose Down
DROP INDEX payment.payments_settled_event_pending_idx;
ALTER TABLE payment.payments
    DROP COLUMN settled_event_pending,
    DROP COLUMN settled_event_traceparent,
    DROP COLUMN settled_event_published_at;
