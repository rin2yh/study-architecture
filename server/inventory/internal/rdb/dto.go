package rdb

import (
	"github.com/rin2yh/study-architecture/server/inventory/internal/db"
)

// port を sqlc 生成型から切り離す (ADR-[[202607011200]])。created_at など API に出さない内部列は
// 載せない。
type StockIn struct {
	ID        int64
	ProductID int64
	Quantity  int32
}

func toStockIn(r db.InventoryStockIn) StockIn {
	return StockIn{
		ID:        r.ID,
		ProductID: r.ProductID,
		Quantity:  r.Quantity,
	}
}
