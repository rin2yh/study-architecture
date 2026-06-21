package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type OrderStub struct {
	Orders []db.OrderOrder
	Order  db.OrderOrder
	Err    error
}

func (s OrderStub) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.Orders, s.Err
}

func (s OrderStub) GetOrder(context.Context, int64) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s OrderStub) CreateOrder(context.Context, db.CreateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s OrderStub) UpdateOrder(context.Context, db.UpdateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}
