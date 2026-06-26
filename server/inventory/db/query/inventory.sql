-- name: StockIn :one
INSERT INTO inventory.movements (product_id, kind, quantity)
VALUES ($1, 'stock_in', $2)
RETURNING id, product_id, kind, quantity, order_id, reservation_id, expires_at, created_at;

-- name: LockProduct :exec
-- 同一商品の同時予約を直列化する。tx 終了まで保持され売り越しを DB で防ぐ (ADR-[[202606262000]])。
SELECT pg_advisory_xact_lock($1::bigint);

-- name: AvailableQty :one
SELECT (
    COALESCE(SUM(m.quantity) FILTER (WHERE m.kind = 'stock_in'), 0)
  - COALESCE(SUM(m.quantity) FILTER (WHERE m.kind = 'confirm'), 0)
  - COALESCE(SUM(m.quantity) FILTER (
        WHERE m.kind = 'reserve'
          AND m.expires_at > now()
          AND NOT EXISTS (SELECT 1 FROM inventory.movements c WHERE c.kind = 'confirm' AND c.reservation_id = m.id)
          AND NOT EXISTS (SELECT 1 FROM inventory.movements x WHERE x.kind = 'release' AND x.reservation_id = m.id)
    ), 0)
)::bigint AS available
FROM inventory.movements m
WHERE m.product_id = $1;

-- name: InsertReservation :one
INSERT INTO inventory.movements (product_id, kind, quantity, order_id, expires_at)
VALUES ($1, 'reserve', $2, $3, now() + ($4::int * interval '1 second'))
RETURNING id;

-- name: ConfirmReservationsByOrder :exec
-- 予約→確定の昇格。payment.settled 再配信は部分ユニークインデックスで吸収する (ADR-[[202606261214]])。
INSERT INTO inventory.movements (product_id, kind, quantity, order_id, reservation_id)
SELECT r.product_id, 'confirm', r.quantity, r.order_id, r.id
FROM inventory.movements r
WHERE r.kind = 'reserve' AND r.order_id = $1
  AND NOT EXISTS (SELECT 1 FROM inventory.movements x WHERE x.kind = 'release' AND x.reservation_id = r.id)
ON CONFLICT (reservation_id) WHERE kind = 'confirm' DO NOTHING;

-- name: ReleaseReservationsByOrder :exec
-- 補償/キャンセル時の解放。未確定の予約だけを反対仕訳で戻す (#88 のフック)。
INSERT INTO inventory.movements (product_id, kind, quantity, order_id, reservation_id)
SELECT r.product_id, 'release', r.quantity, r.order_id, r.id
FROM inventory.movements r
WHERE r.kind = 'reserve' AND r.order_id = $1
  AND NOT EXISTS (SELECT 1 FROM inventory.movements c WHERE c.kind = 'confirm' AND c.reservation_id = r.id)
ON CONFLICT (reservation_id) WHERE kind = 'release' DO NOTHING;

-- name: ReleaseExpiredReservations :exec
-- TTL 切れ予約の遅延回収。worker が定期実行し台帳に解放を追記する (ADR-[[202606262000]])。
INSERT INTO inventory.movements (product_id, kind, quantity, order_id, reservation_id)
SELECT r.product_id, 'release', r.quantity, r.order_id, r.id
FROM inventory.movements r
WHERE r.kind = 'reserve' AND r.expires_at <= now()
  AND NOT EXISTS (SELECT 1 FROM inventory.movements c WHERE c.kind = 'confirm' AND c.reservation_id = r.id)
  AND NOT EXISTS (SELECT 1 FROM inventory.movements x WHERE x.kind = 'release' AND x.reservation_id = r.id)
ON CONFLICT (reservation_id) WHERE kind = 'release' DO NOTHING;
