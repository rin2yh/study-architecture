-- +goose Up
-- 入庫は stock_ins に追記し、予約は reservations の 1 行で表す。確定/解放/期限切れは
-- 書き込み一度きりの nullable タイムスタンプで持ち、状態はどの *_at が入っているかで導出する。
-- 判別列 (kind/status) や在庫数カラムは持たない (ADR-[[202606262000]])。

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
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id   bigint  NOT NULL,
    order_id     bigint  NOT NULL,
    quantity     integer NOT NULL CHECK (quantity > 0),
    created_at   timestamptz NOT NULL DEFAULT now(),
    -- 終端は確定/解放/期限切れの相互排他。各列は NULL から一度だけ時刻が入る。
    confirmed_at timestamptz,
    released_at  timestamptz,
    expired_at   timestamptz,
    CONSTRAINT reservations_one_outcome CHECK (num_nonnulls(confirmed_at, released_at, expired_at) <= 1)
);
CREATE INDEX reservations_product_idx ON inventory.reservations (product_id);
CREATE INDEX reservations_order_idx ON inventory.reservations (order_id);

-- +goose Down
DROP TABLE inventory.reservations;
DROP TABLE inventory.stock_ins;
DROP FUNCTION inventory.reservation_ttl();
