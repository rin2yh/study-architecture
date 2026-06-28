//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewOrderQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(rdb.NewOrderCommand)),
	kessoku.Bind[gateway.ProductPort](kessoku.Provide(gateway.NewProductClient)),
	kessoku.Bind[gateway.PaymentPort](kessoku.Provide(gateway.NewPaymentClient)),
	kessoku.Bind[gateway.InventoryPort](kessoku.Provide(gateway.NewInventoryClient)),
	kessoku.Provide(handler.New),
)
