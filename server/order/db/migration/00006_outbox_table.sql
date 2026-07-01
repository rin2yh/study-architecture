-- +goose Up
-- 集約列方式 (ADR-[[202606261212]]) から専用 outbox テーブルへ移行する (ADR-[[202606300600]] が Supersede)。
CREATE TABLE "order".outbox (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_id bigint      NOT NULL,
    event_type   text        NOT NULL,
    payload      jsonb       NOT NULL,
    traceparent  text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

-- 大半は送信済みで未送信は少数のため。
CREATE INDEX order_outbox_unpublished_idx
    ON "order".outbox (id)
    WHERE published_at IS NULL;

DROP INDEX "order".orders_cancelled_event_pending_idx;
ALTER TABLE "order".orders
    DROP COLUMN cancelled_event_pending,
    DROP COLUMN cancelled_event_traceparent,
    DROP COLUMN cancelled_event_published_at;

-- +goose Down
ALTER TABLE "order".orders
    ADD COLUMN cancelled_event_pending      boolean     NOT NULL DEFAULT false,
    ADD COLUMN cancelled_event_traceparent  text        NOT NULL DEFAULT '',
    ADD COLUMN cancelled_event_published_at timestamptz;

CREATE INDEX orders_cancelled_event_pending_idx
    ON "order".orders (id)
    WHERE cancelled_event_pending;

DROP TABLE "order".outbox;
