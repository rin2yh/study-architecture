package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

// CheckoutLine は確定する 1 明細。product から複写済みのスナップショット値を持つ。
type CheckoutLine struct {
	ProductID      int64
	ProductName    string
	UnitPriceCents int64
	Quantity       int32
}

type OrderRepository interface {
	ListOrders(ctx context.Context) ([]db.OrderOrder, error)
	GetOrder(ctx context.Context, id int64) (db.OrderOrder, error)
	GetOrderItems(ctx context.Context, orderID int64) ([]db.OrderOrderItem, error)
	CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.OrderOrder, error)
	UpdateOrder(ctx context.Context, arg db.UpdateOrderParams) (db.OrderOrder, error)
	Checkout(ctx context.Context, memberID int64, status string, totalCents int64, lines []CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error)
}

type Repository struct {
	pool *pgxpool.Pool
	q    db.Querier
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
	return &Repository{pool: pool, q: db.New(pool)}
}

func (r *Repository) ListOrders(ctx context.Context) ([]db.OrderOrder, error) {
	return r.q.ListOrders(ctx)
}

func (r *Repository) GetOrder(ctx context.Context, id int64) (db.OrderOrder, error) {
	row, err := r.q.GetOrder(ctx, id)
	if err != nil {
		return db.OrderOrder{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *Repository) GetOrderItems(ctx context.Context, orderID int64) ([]db.OrderOrderItem, error) {
	return r.q.ListOrderItems(ctx, orderID)
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

// Checkout は注文ヘッダと明細スナップショットを 1 トランザクションで確定する。明細は
// order 単独で完結する整合 (横断 JOIN なし) なのでローカル tx で原子的に書ける。
// payment / shipping への波及は別サービスのため、この tx の外で呼ぶ ([[0008]])。
func (r *Repository) Checkout(ctx context.Context, memberID int64, status string, totalCents int64, lines []CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.OrderOrder{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	order, err := qtx.CreateOrder(ctx, db.CreateOrderParams{MemberID: memberID, Status: status, TotalCents: totalCents})
	if err != nil {
		return db.OrderOrder{}, nil, err
	}
	items := make([]db.OrderOrderItem, 0, len(lines))
	for _, l := range lines {
		item, err := qtx.CreateOrderItem(ctx, db.CreateOrderItemParams{
			OrderID:        order.ID,
			ProductID:      l.ProductID,
			ProductName:    l.ProductName,
			UnitPriceCents: l.UnitPriceCents,
			Quantity:       l.Quantity,
		})
		if err != nil {
			return db.OrderOrder{}, nil, err
		}
		items = append(items, item)
	}
	if err := tx.Commit(ctx); err != nil {
		return db.OrderOrder{}, nil, err
	}
	return order, items, nil
}
