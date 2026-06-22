//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/payment/internal/event"
	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
	"github.com/rin2yh/study-architecture/server/payment/internal/rdb"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewPaymentQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(rdb.NewPaymentCommand)),
	kessoku.Bind[event.Publisher](kessoku.Provide(event.NewRedisPublisher)),
	kessoku.Provide(handler.New),
)
