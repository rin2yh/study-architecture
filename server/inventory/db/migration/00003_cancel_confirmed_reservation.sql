-- +goose Up
-- 確定済み予約は相互排他の終端 *_at を立てられない (ADR-[[202606262000]])。確定後の取り消しは
-- その制約の外側の cancelled_at で表し、戻しを予約行に閉じる (ADR-[[202606281000]])。
ALTER TABLE inventory.reservations
    ADD COLUMN cancelled_at timestamptz,
    ADD CONSTRAINT reservations_cancel_after_confirm
        CHECK (cancelled_at IS NULL OR confirmed_at IS NOT NULL);

-- +goose Down
ALTER TABLE inventory.reservations
    DROP CONSTRAINT reservations_cancel_after_confirm,
    DROP COLUMN cancelled_at;
