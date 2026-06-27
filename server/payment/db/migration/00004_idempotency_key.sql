-- +goose Up
-- order が checkout 受付時に発番する idempotency key。order→payment POST の再送 (リトライ) で
-- 決済が二重生成されるのを DB ユニーク制約で原子的に弾く (ADR-[[202606261214]])。
-- 既存行と key 未指定の直接作成は '' を入れて重複し得るため、部分ユニークで '' を除外する。
ALTER TABLE payment.payments
    ADD COLUMN idempotency_key text NOT NULL DEFAULT '';

CREATE UNIQUE INDEX payments_idempotency_key_key
    ON payment.payments (idempotency_key)
    WHERE idempotency_key <> '';

-- +goose Down
DROP INDEX payment.payments_idempotency_key_key;
ALTER TABLE payment.payments
    DROP COLUMN idempotency_key;
