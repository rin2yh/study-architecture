-- +goose Up
-- 注文キャンセルの補償イベント (order.cancelled) の Transactional Outbox 用の送信状態列。
-- フォワードの payment.settled (ADR-[[202606261212]]) と対称に、status の cancelled 遷移と
-- 未送信状態を同一トランザクションで確定する (ADR-[[202606261702]])。
ALTER TABLE "order".orders
    ADD COLUMN cancelled_event_pending      boolean     NOT NULL DEFAULT false,
    ADD COLUMN cancelled_event_traceparent  text        NOT NULL DEFAULT '',
    ADD COLUMN cancelled_event_published_at timestamptz;

-- 大半は未キャンセルなので、リレーが走査する未送信行だけを対象にした部分インデックスにする。
CREATE INDEX orders_cancelled_event_pending_idx
    ON "order".orders (id)
    WHERE cancelled_event_pending;

-- +goose Down
DROP INDEX "order".orders_cancelled_event_pending_idx;
ALTER TABLE "order".orders
    DROP COLUMN cancelled_event_pending,
    DROP COLUMN cancelled_event_traceparent,
    DROP COLUMN cancelled_event_published_at;
