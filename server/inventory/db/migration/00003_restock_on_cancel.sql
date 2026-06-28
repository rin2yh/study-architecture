-- +goose Up
-- キャンセル時の在庫戻し。確定済み (confirmed_at) 予約は相互排他 CHECK のため released_at を
-- 立てられないので、戻しを stock_ins への補償追記 (+quantity) で表す (反対仕訳。ADR-[[202606281000]])。
-- 用途を名前で限定し (reservations の *_at と同じく case を列名で表す)、再配信での二重戻しを
-- 部分ユニークで弾く冪等キーにする (ADR-[[202606261214]])。通常入庫や他用途は別列/NULL で表す。
ALTER TABLE inventory.stock_ins
    ADD COLUMN cancelled_reservation_id bigint REFERENCES inventory.reservations (id);

CREATE UNIQUE INDEX stock_ins_cancelled_reservation_id_key
    ON inventory.stock_ins (cancelled_reservation_id)
    WHERE cancelled_reservation_id IS NOT NULL;

-- +goose Down
DROP INDEX inventory.stock_ins_cancelled_reservation_id_key;
ALTER TABLE inventory.stock_ins
    DROP COLUMN cancelled_reservation_id;
