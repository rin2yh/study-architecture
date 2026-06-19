package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type OrderCommand struct {
	q db.Querier
}

func NewOrderCommand(pool *pgxpool.Pool) *OrderCommand {
	return &OrderCommand{q: db.New(pool)}
}

func (r *OrderCommand) CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.OrderOrder, error) {
	return r.q.CreateOrder(ctx, arg)
}

func (r *OrderCommand) UpdateOrder(ctx context.Context, arg db.UpdateOrderParams) (db.OrderOrder, error) {
	row, err := r.q.UpdateOrder(ctx, arg)
	if err != nil {
		return db.OrderOrder{}, dberr.FromUpdate(err)
	}
	return row, nil
}
