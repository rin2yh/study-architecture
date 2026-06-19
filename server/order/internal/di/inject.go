//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(repository.NewOrderQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(repository.NewOrderCommand)),
	kessoku.Provide(handler.New),
)
