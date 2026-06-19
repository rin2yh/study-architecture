//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
	"github.com/rin2yh/study-architecture/server/payment/internal/repository"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(repository.NewPaymentQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(repository.NewPaymentCommand)),
	kessoku.Provide(handler.New),
)
