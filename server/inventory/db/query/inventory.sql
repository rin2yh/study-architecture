-- name: StockIn :one
INSERT INTO inventory.stock_ins (product_id, quantity)
VALUES ($1, $2)
RETURNING id, product_id, quantity, created_at;

-- name: LockProduct :exec
-- 同一商品の同時予約を直列化する。tx 終了まで保持され売り越しを DB で防ぐ (ADR-[[202606262000]])。
SELECT pg_advisory_xact_lock($1::bigint);

-- name: AvailableQty :one
-- 在庫数を保存せず集計で導く (ADR-[[202606262000]])。
SELECT COALESCE(SUM(d.delta), 0)::bigint AS available
FROM (
    SELECT s.quantity::bigint AS delta
    FROM inventory.stock_ins s
    WHERE s.product_id = $1
  UNION ALL
    SELECT -r.quantity::bigint
    FROM inventory.reservations r
    WHERE r.product_id = $1 AND r.confirmed_at IS NOT NULL
  UNION ALL
    SELECT -r.quantity::bigint
    FROM inventory.reservations r
    WHERE r.product_id = $1
      AND r.confirmed_at IS NULL AND r.released_at IS NULL AND r.expired_at IS NULL
      AND r.created_at + inventory.reservation_ttl() > now()
) d;

-- name: InsertReservation :one
INSERT INTO inventory.reservations (product_id, order_id, quantity)
VALUES ($1, $2, $3)
RETURNING id;

-- name: ConfirmReservationsByOrder :exec
-- 終端 *_at が未設定の予約だけ確定する。payment.settled 再配信は 0 行更新で吸収 (ADR-[[202606261214]])。
UPDATE inventory.reservations
SET confirmed_at = now()
WHERE order_id = $1 AND confirmed_at IS NULL AND released_at IS NULL AND expired_at IS NULL;

-- name: ReleaseReservationsByOrder :exec
-- 補償/キャンセル時の解放 (#88 のフック)。
UPDATE inventory.reservations
SET released_at = now()
WHERE order_id = $1 AND confirmed_at IS NULL AND released_at IS NULL AND expired_at IS NULL;

-- name: ExpireReservations :exec
-- (ADR-[[202606262000]])
UPDATE inventory.reservations
SET expired_at = now()
WHERE confirmed_at IS NULL AND released_at IS NULL AND expired_at IS NULL
  AND created_at + inventory.reservation_ttl() <= now();
