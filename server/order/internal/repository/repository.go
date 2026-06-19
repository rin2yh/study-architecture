package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type OrderQuery struct {
	q db.Querier
}

type OrderCommand struct {
	q db.Querier
}

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

func NewOrderQuery(pool *pgxpool.Pool) *OrderQuery {
	return &OrderQuery{q: db.New(pool)}
}

func NewOrderCommand(pool *pgxpool.Pool) *OrderCommand {
	return &OrderCommand{q: db.New(pool)}
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
