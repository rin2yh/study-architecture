//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/internal/redisx"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
)

var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Provide(redisx.NewClient),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewOrderQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(rdb.NewOrderCommand)),
	kessoku.Bind[gateway.ProductPort](kessoku.Provide(gateway.NewProductClient)),
	kessoku.Bind[gateway.PaymentPort](kessoku.Provide(gateway.NewPaymentClient)),
	kessoku.Bind[gateway.InventoryPort](kessoku.Provide(gateway.NewInventoryClient)),
	kessoku.Bind[outbox.Store](kessoku.Provide(rdb.NewOutboxStore)),
	kessoku.Provide(outbox.NewRelay),
	kessoku.Provide(handler.New),
	kessoku.Provide(NewApp),
)
