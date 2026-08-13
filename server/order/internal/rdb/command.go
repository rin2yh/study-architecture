package rdb

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

// sqlc 生成型を共有層に持ち込まないためのアダプタ。
type outboxInserter struct{ q db.Querier }

func (o outboxInserter) InsertOutbox(ctx context.Context, row outbox.Row) error {
	aggregateID, err := toInt64(row.AggregateID)
	if err != nil {
		return err
	}
	return o.q.InsertOutbox(ctx, db.InsertOutboxParams{
		AggregateID: aggregateID,
		EventType:   row.EventType,
		Payload:     row.Payload,
		Traceparent: row.Traceparent,
	})
}

// 発送済み注文はキャンセル不可で返品フローへ分岐する (ADR-[[202606261702]])。handler は 409 に対応づける。
var ErrNotCancellable = errors.New("order not cancellable")

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

// キャンセル可否の判定と遷移を 1 tx で直列化する (ADR-[[202606261702]])。
func (r *OrderCommand) CancelOrder(ctx context.Context, id int64, traceparent string) (db.OrderOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.OrderOrder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := db.New(tx)
	current, err := qtx.GetOrderForUpdate(ctx, id)
	if err != nil {
		return db.OrderOrder{}, dberr.FromRead(err)
	}
	switch current.Status {
	case "shipped":
		return db.OrderOrder{}, ErrNotCancellable
	case "cancelled":
		return current, nil
	}
	cancelled, err := qtx.CancelOrder(ctx, id)
	if err != nil {
		return db.OrderOrder{}, err
	}
	orderID, err := order.Parse(strconv.FormatInt(cancelled.ID, 10))
	if err != nil {
		return db.OrderOrder{}, err
	}
	if err := outbox.Dispatch(ctx, outboxInserter{qtx}, traceparent, orderevent.Cancelled{OrderID: orderID}); err != nil {
		return db.OrderOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.OrderOrder{}, err
	}
	return cancelled, nil
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

// ID を数値へ戻すのはここだけ。列と生成コードが bigint / int64 なのはこの層の事情で、
// ドメインの order.ID は表現を持たない。
func toInt64(raw string) (int64, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", raw, err)
	}
	return v, nil
}
