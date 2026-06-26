-- +goose Up
-- 利用可能在庫を集計で導く単一情報源。在庫数カラムを持たない (ADR-[[202606262000]])。
CREATE TABLE inventory.movements (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- product を引かずスナップショット参照で持つ (ADR-[[202606190900]])。
    product_id     bigint  NOT NULL,
    kind           text    NOT NULL CHECK (kind IN ('stock_in', 'reserve', 'confirm', 'release')),
    quantity       integer NOT NULL CHECK (quantity > 0),
    order_id       bigint,
    reservation_id bigint REFERENCES inventory.movements(id),
    -- 期限切れは worker が解放で回収する (ADR-[[202606262000]])。
    expires_at     timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX movements_product_idx ON inventory.movements (product_id);
CREATE INDEX movements_open_reserve_idx ON inventory.movements (order_id) WHERE kind = 'reserve';

-- 予約 1 行につき確定・解放は高々 1 行。payment.settled の再配信や二重解放を
-- ON CONFLICT DO NOTHING で原子的に吸収する (ADR-[[202606261214]] と同型)。
CREATE UNIQUE INDEX movements_one_confirm_per_reservation ON inventory.movements (reservation_id) WHERE kind = 'confirm';
CREATE UNIQUE INDEX movements_one_release_per_reservation ON inventory.movements (reservation_id) WHERE kind = 'release';

-- +goose Down
DROP TABLE inventory.movements;
