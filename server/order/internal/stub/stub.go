package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type Repo struct {
	Orders   []db.OrderOrder
	ByMember []db.OrderOrder
	Order    db.OrderOrder
	Err      error
}

func (s Repo) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.Orders, s.Err
}

func (s Repo) ListOrdersByMember(context.Context, int64) ([]db.OrderOrder, error) {
	return s.ByMember, s.Err
}

func (s Repo) GetOrder(context.Context, int64) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s Repo) CreateOrder(context.Context, db.CreateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}

func (s Repo) UpdateOrder(context.Context, db.UpdateOrderParams) (db.OrderOrder, error) {
	return s.Order, s.Err
}
