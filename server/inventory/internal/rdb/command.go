package rdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/inventory/internal/db"
)

// 致命扱いの在庫不足 (ADR-[[202606261216]])。handler は 409 に対応づける。
var ErrInsufficientStock = errors.New("insufficient stock")

type ReserveLine struct {
	ProductID int64
	Quantity  int32
}

type InventoryCommand struct {
	pool *pgxpool.Pool
	q    db.Querier
}

func NewInventoryCommand(pool *pgxpool.Pool) *InventoryCommand {
	return &InventoryCommand{pool: pool, q: db.New(pool)}
}

func (r *InventoryCommand) StockIn(ctx context.Context, productID int64, quantity int32) (db.InventoryStockIn, error) {
	row, err := r.q.StockIn(ctx, db.StockInParams{ProductID: productID, Quantity: quantity})
	if err != nil {
		return db.InventoryStockIn{}, dberr.FromWrite(err)
	}
	return row, nil
}

// (ADR-[[202606262000]])
func (r *InventoryCommand) Reserve(ctx context.Context, orderID int64, lines []ReserveLine) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	for _, l := range lines {
		if err := qtx.LockProduct(ctx, l.ProductID); err != nil {
			return err
		}
		available, err := qtx.AvailableQty(ctx, l.ProductID)
		if err != nil {
			return err
		}
		if available < int64(l.Quantity) {
			return ErrInsufficientStock
		}
		if _, err := qtx.InsertReservation(ctx, db.InsertReservationParams{
			ProductID: l.ProductID,
			OrderID:   orderID,
			Quantity:  l.Quantity,
		}); err != nil {
			return dberr.FromWrite(err)
		}
	}
	return tx.Commit(ctx)
}

func (r *InventoryCommand) ConfirmReservationsByOrder(ctx context.Context, orderID int64) error {
	return r.q.ConfirmReservationsByOrder(ctx, orderID)
}

func (r *InventoryCommand) ReleaseReservationsByOrder(ctx context.Context, orderID int64) error {
	return r.q.ReleaseReservationsByOrder(ctx, orderID)
}

// CompensateByOrder は order.cancelled の補償。未確定は解放・確定済みは反対仕訳の補償 stock_in で戻す。
// 状態ガードと reservation_id ユニークで冪等にし、再配信での二重戻しを防ぐ (ADR-[[202606281000]])。
func (r *InventoryCommand) CompensateByOrder(ctx context.Context, orderID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	if err := qtx.ReleaseReservationsByOrder(ctx, orderID); err != nil {
		return err
	}
	if err := qtx.RestockConfirmedReservationsByOrder(ctx, orderID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *InventoryCommand) ExpireReservations(ctx context.Context) error {
	return r.q.ExpireReservations(ctx)
}
