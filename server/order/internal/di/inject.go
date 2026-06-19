//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[repository.OrderRepository](kessoku.Provide(repository.NewRepository)),
	kessoku.Bind[gateway.ProductPort](kessoku.Provide(gateway.NewProductClient)),
	kessoku.Bind[gateway.PaymentPort](kessoku.Provide(gateway.NewPaymentClient)),
	kessoku.Provide(handler.New),
)
