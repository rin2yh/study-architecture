//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/internal/redisx"
	"github.com/rin2yh/study-architecture/server/shipping/internal/consumer"
	"github.com/rin2yh/study-architecture/server/shipping/internal/handler"
	"github.com/rin2yh/study-architecture/server/shipping/internal/rdb"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewShipmentQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(rdb.NewShipmentCommand)),
	kessoku.Provide(handler.New),
)

var _ = kessoku.Inject[*consumer.Consumer](
	"InitConsumer",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Provide(redisx.NewClient),
	kessoku.Bind[consumer.ShipmentCreator](kessoku.Provide(rdb.NewShipmentCommand)),
	kessoku.Provide(consumer.New),
)
