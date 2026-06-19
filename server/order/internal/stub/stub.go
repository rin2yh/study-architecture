package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
)

type Repo struct {
	Orders []db.OrderOrder
	Order  db.OrderOrder
	Items  []db.OrderOrderItem
	Err    error
}

func (s Repo) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.Orders, s.Err
}

func (s Repo) GetOrder(context.Context, int64) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s Repo) GetOrderItems(context.Context, int64) ([]db.OrderOrderItem, error) {
	return s.Items, s.Err
}

func (s Repo) CreateOrder(context.Context, db.CreateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s Repo) UpdateOrder(context.Context, db.UpdateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s Repo) Checkout(context.Context, int64, string, int64, []repository.CheckoutLine) (db.OrderOrder, []db.OrderOrderItem, error) {
	return s.Order, s.Items, s.Err
}

// Product は ProductPort のテスト用スタブ。
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

// Payment は PaymentPort のテスト用スタブ。
type Payment struct {
	ID  int64
	Err error
}

func (s Payment) CreatePayment(context.Context, int64, int64, string) (int64, error) {
	return s.ID, s.Err
}
