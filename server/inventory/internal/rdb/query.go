package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/inventory/internal/db"
)

type InventoryQuery struct {
	q db.Querier
}

func NewInventoryQuery(pool *pgxpool.Pool) *InventoryQuery {
	return &InventoryQuery{q: db.New(pool)}
}

func (r *InventoryQuery) Available(ctx context.Context, productID int64) (int64, error) {
	return r.q.AvailableQty(ctx, productID)
}
