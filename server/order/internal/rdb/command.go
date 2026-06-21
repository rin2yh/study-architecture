package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type CheckoutLine struct {
	ProductID      int64
	ProductName    string
	UnitPriceCents int64
	Quantity       int32
}

type OrderCommand struct {
	pool *pgxpool.Pool
	q    db.Querier
}

func NewOrderCommand(pool *pgxpool.Pool) *OrderCommand {
	return &OrderCommand{pool: pool, q: db.New(pool)}
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

func (r *OrderCommand) Checkout(ctx context.Context, memberID int64, status string, totalCents int64, lines []CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error) {
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
