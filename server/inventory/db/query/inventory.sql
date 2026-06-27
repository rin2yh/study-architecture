-- name: StockIn :one
INSERT INTO inventory.stock_ins (product_id, quantity)
VALUES ($1, $2)
RETURNING id, product_id, quantity, created_at;

-- name: LockProduct :exec
-- 同一商品の同時予約を直列化する。tx 終了まで保持され売り越しを DB で防ぐ (ADR-[[202606262000]])。
SELECT pg_advisory_xact_lock($1::bigint);

-- name: AvailableQty :one
-- 利用可能在庫 = 入庫(+) と、確定・未確定で生きている予約(-) の符号付き合計。
SELECT COALESCE(SUM(d.delta), 0)::bigint AS available
FROM (
    SELECT s.quantity::bigint AS delta
    FROM inventory.stock_ins s
    WHERE s.product_id = $1
  UNION ALL
    SELECT -r.quantity::bigint
    FROM inventory.reservations r
    JOIN inventory.confirmations c ON c.reservation_id = r.id
    WHERE r.product_id = $1
  UNION ALL
    SELECT -r.quantity::bigint
    FROM inventory.reservations r
    WHERE r.product_id = $1 AND r.created_at + inventory.reservation_ttl() > now()
      AND NOT EXISTS (SELECT 1 FROM inventory.confirmations c WHERE c.reservation_id = r.id)
      AND NOT EXISTS (SELECT 1 FROM inventory.releases x WHERE x.reservation_id = r.id)
) d;

-- name: InsertReservation :one
INSERT INTO inventory.reservations (product_id, order_id, quantity)
VALUES ($1, $2, $3)
RETURNING id;

-- name: ConfirmReservationsByOrder :exec
-- payment.settled 再配信は主キー衝突で吸収する (ADR-[[202606261214]])。終端済み (解放/期限切れ) は確定しない。
INSERT INTO inventory.confirmations (reservation_id)
SELECT r.id FROM inventory.reservations r
WHERE r.order_id = $1
  AND NOT EXISTS (SELECT 1 FROM inventory.releases x WHERE x.reservation_id = r.id)
  AND NOT EXISTS (SELECT 1 FROM inventory.expirations e WHERE e.reservation_id = r.id)
ON CONFLICT (reservation_id) DO NOTHING;

-- name: ReleaseReservationsByOrder :exec
-- 補償/キャンセル時の解放 (#88 のフック)。終端済み (確定/期限切れ) は解放しない。
INSERT INTO inventory.releases (reservation_id)
SELECT r.id FROM inventory.reservations r
WHERE r.order_id = $1
  AND NOT EXISTS (SELECT 1 FROM inventory.confirmations c WHERE c.reservation_id = r.id)
  AND NOT EXISTS (SELECT 1 FROM inventory.expirations e WHERE e.reservation_id = r.id)
ON CONFLICT (reservation_id) DO NOTHING;

-- name: ExpireReservations :exec
-- TTL 期限切れの回収。意図的な解放 (releases) と区別して expirations に記録する (ADR-[[202606262000]])。
INSERT INTO inventory.expirations (reservation_id)
SELECT r.id FROM inventory.reservations r
WHERE r.created_at + inventory.reservation_ttl() <= now()
  AND NOT EXISTS (SELECT 1 FROM inventory.confirmations c WHERE c.reservation_id = r.id)
  AND NOT EXISTS (SELECT 1 FROM inventory.releases x WHERE x.reservation_id = r.id)
ON CONFLICT (reservation_id) DO NOTHING;
