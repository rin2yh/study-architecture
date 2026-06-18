package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type Repo struct {
	Orders []db.OrderOrder
	Err    error
}

func (s Repo) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return s.Orders, s.Err
}
