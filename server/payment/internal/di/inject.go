//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/internal/redisx"
	"github.com/rin2yh/study-architecture/server/payment/internal/consumer"
	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
	"github.com/rin2yh/study-architecture/server/payment/internal/rdb"
)

// kessoku は同一 concrete の二重 provide を許さない。
var _ = kessoku.Inject[*App](
	"InitApp",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Provide(redisx.NewClient),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewPaymentQuery)),
	kessoku.Provide(rdb.NewPaymentCommand),
	kessoku.Provide(func(c *rdb.PaymentCommand) handler.Command { return c }),
	kessoku.Provide(func(c *rdb.PaymentCommand) consumer.PaymentRefunder { return c }),
	kessoku.Bind[outbox.Store](kessoku.Provide(rdb.NewOutboxStore)),
	kessoku.Provide(outbox.NewRelay),
	kessoku.Provide(consumer.New),
	kessoku.Provide(handler.New),
	kessoku.Provide(NewApp),
)
