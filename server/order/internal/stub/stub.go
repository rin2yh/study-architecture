package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
)

type OrderStub struct {
	Orders    []db.OrderOrder
	ByMember  []db.OrderOrder
	Order     db.OrderOrder
	Items     []db.OrderOrderItem
	Err       error
	DeleteErr error
	CancelErr error
}

func (s OrderStub) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.Orders, s.Err
}

func (s OrderStub) ListOrdersByMember(context.Context, int64) ([]db.OrderOrder, error) {
	return s.ByMember, s.Err
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

func (s OrderStub) DeleteOrder(context.Context, int64) error {
	return s.DeleteErr
}

func (s OrderStub) CancelOrder(context.Context, int64, string) (db.OrderOrder, error) {
	return s.Order, s.CancelErr
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

func TwoProducts() Product {
	return Product{Snapshots: map[int64]gateway.ProductSnapshot{
		100: {ID: 100, Name: "Widget", UnitPriceCents: 500},
		200: {ID: 200, Name: "Gadget", UnitPriceCents: 1500},
	}}
}

type Payment struct {
	ID  int64
	Err error
}

func (s Payment) CreatePayment(context.Context, int64, int64, string, string) (int64, error) {
	return s.ID, s.Err
}

type Inventory struct {
	ReserveErr error
	ReleaseErr error
}

func (s Inventory) Reserve(context.Context, int64, []gateway.ReserveLine) error {
	return s.ReserveErr
}

func (s Inventory) Release(context.Context, int64) error {
	return s.ReleaseErr
}
