-- +goose Up
-- 利用可能在庫を集計で導く append-only 台帳。在庫数カラムを持たず、変動を種別ごとの
-- テーブルへ追記する。判別列 (kind) は持たない (ADR-[[202606262000]])。

-- 予約の取り置き有効期間。expires_at を行に保存せず created_at からの導出に使う唯一の出所
-- (在庫数・status と同じく導出できる値は持たない。ADR-[[202606262000]])。
-- +goose StatementBegin
CREATE FUNCTION inventory.reservation_ttl() RETURNS interval
    LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$ SELECT interval '15 minutes' $$;
-- +goose StatementEnd

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
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reservations_product_idx ON inventory.reservations (product_id);
CREATE INDEX reservations_order_idx ON inventory.reservations (order_id);

-- 予約の終端状態。reservation_id を主キーにすることで、再配信や二重適用を DB が原子的に弾く
-- (ADR-[[202606261214]])。確定・解放・期限切れは予約 1 件につき相互排他で各テーブル高々 1 行。
-- 種別ごとにテーブルを分け、判別列 (reason 等の enum) を持ち込まない (ADR-[[202606262000]])。
CREATE TABLE inventory.confirmations (
    reservation_id bigint PRIMARY KEY REFERENCES inventory.reservations(id),
    created_at     timestamptz NOT NULL DEFAULT now()
);
-- 意図的な解放 (補償・キャンセル #88)。
CREATE TABLE inventory.releases (
    reservation_id bigint PRIMARY KEY REFERENCES inventory.reservations(id),
    created_at     timestamptz NOT NULL DEFAULT now()
);
-- (ADR-[[202606262000]])
CREATE TABLE inventory.expirations (
    reservation_id bigint PRIMARY KEY REFERENCES inventory.reservations(id),
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE inventory.expirations;
DROP TABLE inventory.releases;
DROP TABLE inventory.confirmations;
DROP TABLE inventory.reservations;
DROP TABLE inventory.stock_ins;
DROP FUNCTION inventory.reservation_ttl();
