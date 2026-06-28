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

// CheckoutAddress は注文時点の配送先スナップショット (ADR-[[202606261704]])。
type CheckoutAddress struct {
	Recipient  string
	PostalCode string
	Prefecture string
	City       string
	Line1      string
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

// DeleteOrder は予約失敗時の補償で注文を取り消す。order_items は ON DELETE CASCADE で連れて消える。
func (r *OrderCommand) DeleteOrder(ctx context.Context, id int64) error {
	return r.q.DeleteOrder(ctx, id)
}

func (r *OrderCommand) Checkout(ctx context.Context, memberID int64, status string, totalCents int64, lines []CheckoutLine, addr CheckoutAddress) (db.OrderOrder, []db.OrderOrderItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.OrderOrder{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	order, err := qtx.CreateOrderWithShipping(ctx, db.CreateOrderWithShippingParams{
		MemberID:           memberID,
		Status:             status,
		TotalCents:         totalCents,
		ShippingRecipient:  addr.Recipient,
		ShippingPostalCode: addr.PostalCode,
		ShippingPrefecture: addr.Prefecture,
		ShippingCity:       addr.City,
		ShippingLine1:      addr.Line1,
	})
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
