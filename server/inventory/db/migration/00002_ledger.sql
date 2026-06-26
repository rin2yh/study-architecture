-- +goose Up
-- 利用可能在庫を集計で導く append-only 台帳。在庫数カラムを持たず、変動を種別ごとの
-- テーブルへ追記する。判別列 (kind) は持たない (ADR-[[202606262000]])。
CREATE TABLE inventory.stock_ins (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- product を引かずスナップショット参照で持つ (ADR-[[202606190900]])。
    product_id bigint  NOT NULL,
    quantity   integer NOT NULL CHECK (quantity > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX stock_ins_product_idx ON inventory.stock_ins (product_id);

CREATE TABLE inventory.reservations (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint  NOT NULL,
    order_id   bigint  NOT NULL,
    quantity   integer NOT NULL CHECK (quantity > 0),
    -- 期限切れは worker が解放で回収する (ADR-[[202606262000]])。
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reservations_product_idx ON inventory.reservations (product_id);
CREATE INDEX reservations_order_idx ON inventory.reservations (order_id);

-- 予約の終端状態。reservation_id を主キーにすることで、再配信や二重適用を DB が原子的に弾く
-- (ADR-[[202606261214]])。確定と解放は予約 1 件につき相互排他で高々 1 行。
CREATE TABLE inventory.confirmations (
    reservation_id bigint PRIMARY KEY REFERENCES inventory.reservations(id),
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE inventory.releases (
    reservation_id bigint PRIMARY KEY REFERENCES inventory.reservations(id),
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE inventory.releases;
DROP TABLE inventory.confirmations;
DROP TABLE inventory.reservations;
DROP TABLE inventory.stock_ins;
