package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
)

type OrderStub struct {
	Orders []db.OrderOrder
	Order  db.OrderOrder
	Items  []db.OrderOrderItem
	Err    error
}

func (s OrderStub) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.Orders, s.Err
}

func (s OrderStub) GetOrder(context.Context, int64) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s OrderStub) GetOrderItems(context.Context, int64) ([]db.OrderOrderItem, error) {
	return s.Items, s.Err
}

func (s OrderStub) CreateOrder(context.Context, db.CreateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s OrderStub) UpdateOrder(context.Context, db.UpdateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s OrderStub) Checkout(context.Context, int64, string, int64, []rdb.CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error) {
	return s.Order, s.Items, s.Err
}

type CheckoutRecorder struct {
	OrderStub
	Err      error
	GotTotal int64
	GotLines []rdb.CheckoutLine
}

func (s *CheckoutRecorder) Checkout(_ context.Context, memberID int64, status string, total int64, lines []rdb.CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error) {
	s.GotTotal = total
	s.GotLines = lines
	if s.Err != nil {
		return db.OrderOrder{}, nil, s.Err
	}
	order := db.OrderOrder{ID: 7, MemberID: memberID, Status: status, TotalCents: total}
	items := make([]db.OrderOrderItem, 0, len(lines))
	for i, l := range lines {
		items = append(items, db.OrderOrderItem{
			ID: int64(i + 1), OrderID: order.ID, ProductID: l.ProductID,
			ProductName: l.ProductName, UnitPriceCents: l.UnitPriceCents, Quantity: l.Quantity,
		})
	}
	return order, items, nil
}

type Product struct {
	Snapshots map[int64]gateway.ProductSnapshot
	Snapshot  gateway.ProductSnapshot
	Err       error
}

func (s Product) FetchProduct(_ context.Context, id int64) (gateway.ProductSnapshot, error) {
	if s.Err != nil {
		return gateway.ProductSnapshot{}, s.Err
	}
	if snap, ok := s.Snapshots[id]; ok {
		return snap, nil
	}
	return s.Snapshot, nil
}

type Payment struct {
	ID  int64
	Err error
}

func (s Payment) CreatePayment(context.Context, int64, int64, string) (int64, error) {
	return s.ID, s.Err
}
