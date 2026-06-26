// Package stub は handler テスト用に Query / Command の最小実装を提供する。
package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/inventory/internal/db"
	"github.com/rin2yh/study-architecture/server/inventory/internal/rdb"
)

type InventoryStub struct {
	AvailableQty int64
	StockInRow   db.InventoryStockIn
	ReserveErr   error
	Err          error
}

func (s InventoryStub) Available(context.Context, int64) (int64, error) {
	return s.AvailableQty, s.Err
}

func (s InventoryStub) StockIn(context.Context, int64, int32) (db.InventoryStockIn, error) {
	return s.StockInRow, s.Err
}

func (s InventoryStub) Reserve(context.Context, int64, []rdb.ReserveLine, int32) error {
	return s.ReserveErr
}

func (s InventoryStub) ReleaseReservationsByOrder(context.Context, int64) error {
	return s.Err
}
