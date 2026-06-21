package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type OrderRepository interface {
	ListOrders(ctx context.Context) ([]db.OrderOrder, error)
	ListOrdersByMember(ctx context.Context, memberID int64) ([]db.OrderOrder, error)
	GetOrder(ctx context.Context, id int64) (db.OrderOrder, error)
	CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.OrderOrder, error)
	UpdateOrder(ctx context.Context, arg db.UpdateOrderParams) (db.OrderOrder, error)
}

type Repository struct {
	q db.Querier
}

var _ OrderRepository = (*Repository)(nil)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: db.New(pool)}
}

func (r *Repository) ListOrders(ctx context.Context) ([]db.OrderOrder, error) {
	return r.q.ListOrders(ctx)
}

func (r *Repository) ListOrdersByMember(ctx context.Context, memberID int64) ([]db.OrderOrder, error) {
	return r.q.ListOrdersByMember(ctx, memberID)
}

func (r *Repository) GetOrder(ctx context.Context, id int64) (db.OrderOrder, error) {
	row, err := r.q.GetOrder(ctx, id)
	if err != nil {
		return db.OrderOrder{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *Repository) CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.OrderOrder, error) {
	return r.q.CreateOrder(ctx, arg)
}

func (r *Repository) UpdateOrder(ctx context.Context, arg db.UpdateOrderParams) (db.OrderOrder, error) {
	row, err := r.q.UpdateOrder(ctx, arg)
	if err != nil {
		return db.OrderOrder{}, dberr.FromUpdate(err)
	}
	return row, nil
}
