//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/sqsx"
	"github.com/rin2yh/study-architecture/server/shipping/internal/consumer"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
	"github.com/rin2yh/study-architecture/server/shipping/internal/handler"
	"github.com/rin2yh/study-architecture/server/shipping/internal/rdb"
	"github.com/rin2yh/study-architecture/server/shipping/internal/worker"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewShipmentQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(rdb.NewShipmentCommand)),
	kessoku.Provide(handler.New),
)

// kessoku は同一 concrete の二重 provide を許さない。
var _ = kessoku.Inject[*worker.Worker](
	"InitWorker",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Provide(sqsx.NewClient),
	kessoku.Provide(func(c *sqsx.Client) messaging.Subscriber { return c }),
	kessoku.Provide(rdb.NewShipmentCommand),
	kessoku.Provide(func(c *rdb.ShipmentCommand) consumer.ShipmentCreator { return c }),
	kessoku.Provide(func(c *rdb.ShipmentCommand) consumer.ShipmentCanceller { return c }),
	kessoku.Bind[gateway.OrderPort](kessoku.Provide(gateway.NewOrderClient)),
	kessoku.Provide(consumer.New),
	kessoku.Provide(consumer.NewCancel),
	kessoku.Provide(worker.New),
)
