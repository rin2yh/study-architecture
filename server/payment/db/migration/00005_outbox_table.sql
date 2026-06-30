-- +goose Up
-- 集約列方式 (ADR-[[202606261212]]) から専用 outbox テーブルへ移行する (ADR-[[202606300600]] が Supersede)。
CREATE TABLE payment.outbox (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_id bigint      NOT NULL,
    event_type   text        NOT NULL,
    payload      jsonb       NOT NULL,
    traceparent  text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

-- 大半は送信済みで未送信は少数のため。
CREATE INDEX payment_outbox_unpublished_idx
    ON payment.outbox (id)
    WHERE published_at IS NULL;

-- 直後に列を落とすため、先に移さないと in-flight の未送信が失われる。
-- payload のキーは paymentevent.Settled.Values() と一致させる。
INSERT INTO payment.outbox (aggregate_id, event_type, payload, traceparent)
SELECT id,
       'payment.settled',
       jsonb_build_object('event', 'payment.settled', 'paymentId', id, 'orderId', order_id, 'amountCents', amount_cents),
       settled_event_traceparent
FROM payment.payments
WHERE settled_event_pending;

DROP INDEX payment.payments_settled_event_pending_idx;
ALTER TABLE payment.payments
    DROP COLUMN settled_event_pending,
    DROP COLUMN settled_event_traceparent,
    DROP COLUMN settled_event_published_at;

-- +goose Down
ALTER TABLE payment.payments
    ADD COLUMN settled_event_pending      boolean     NOT NULL DEFAULT false,
    ADD COLUMN settled_event_traceparent  text        NOT NULL DEFAULT '',
    ADD COLUMN settled_event_published_at timestamptz;

CREATE INDEX payments_settled_event_pending_idx
    ON payment.payments (id)
    WHERE settled_event_pending;

DROP TABLE payment.outbox;
