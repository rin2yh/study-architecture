-- +goose Up
-- carrier / tracking_no は配送手配 (preparing 枠作成) の時点では未確定で、運送会社割当後に
-- 埋まる。確定時の業務事実ではないので作成時必須にせず DEFAULT '' で「未割当」を表す
-- (ADR-[[202606211200]])。order_id は 1 注文 1 配送の冪等キー (イベント再配送での重複防止)。
CREATE TABLE shipping.shipments (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id    bigint NOT NULL UNIQUE,
    carrier     text NOT NULL DEFAULT '',
    tracking_no text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'preparing',
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE shipping.shipments;
