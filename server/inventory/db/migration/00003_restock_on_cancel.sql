-- +goose Up
-- キャンセル時の在庫戻し。確定済み (confirmed_at) 予約は相互排他 CHECK のため released_at を
-- 立てられないので、戻しを stock_ins への補償追記 (+quantity) で表す (反対仕訳。ADR-[[202606281000]])。
-- どの予約の戻しかを reservation_id で辿れるようにし (トレーサビリティ)、同時に再配信での
-- 二重戻しを部分ユニークで弾く冪等キーにする (ADR-[[202606261214]])。
ALTER TABLE inventory.stock_ins
    ADD COLUMN reservation_id bigint REFERENCES inventory.reservations (id);

CREATE UNIQUE INDEX stock_ins_reservation_id_key
    ON inventory.stock_ins (reservation_id)
    WHERE reservation_id IS NOT NULL;

-- +goose Down
DROP INDEX inventory.stock_ins_reservation_id_key;
ALTER TABLE inventory.stock_ins
    DROP COLUMN reservation_id;
