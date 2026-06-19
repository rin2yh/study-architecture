package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type OrderQuery struct {
	q db.Querier
}

func NewOrderQuery(pool *pgxpool.Pool) *OrderQuery {
	return &OrderQuery{q: db.New(pool)}
}

func (r *OrderQuery) ListOrders(ctx context.Context) ([]db.OrderOrder, error) {
	return r.q.ListOrders(ctx)
}

func (r *OrderQuery) GetOrder(ctx context.Context, id int64) (db.OrderOrder, error) {
	row, err := r.q.GetOrder(ctx, id)
	if err != nil {
		return db.OrderOrder{}, dberr.FromRead(err)
	}
	return row, nil
}
