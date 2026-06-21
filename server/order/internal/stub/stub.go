package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type RDB struct {
	Orders []db.OrderOrder
	Order  db.OrderOrder
	Err    error
}

func (s RDB) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.Orders, s.Err
}

func (s RDB) GetOrder(context.Context, int64) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s RDB) CreateOrder(context.Context, db.CreateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s RDB) UpdateOrder(context.Context, db.UpdateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}
