//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/internal/redisx"
	"github.com/rin2yh/study-architecture/server/inventory/internal/consumer"
	"github.com/rin2yh/study-architecture/server/inventory/internal/handler"
	"github.com/rin2yh/study-architecture/server/inventory/internal/rdb"
	"github.com/rin2yh/study-architecture/server/inventory/internal/reaper"
	"github.com/rin2yh/study-architecture/server/inventory/internal/worker"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewInventoryQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(rdb.NewInventoryCommand)),
	kessoku.Provide(handler.New),
)

// kessoku は同一 concrete の二重 provide を許さない。
var _ = kessoku.Inject[*worker.Worker](
	"InitWorker",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Provide(redisx.NewClient),
	kessoku.Provide(rdb.NewInventoryCommand),
	kessoku.Provide(func(c *rdb.InventoryCommand) consumer.ReservationConfirmer { return c }),
	kessoku.Provide(func(c *rdb.InventoryCommand) reaper.ReservationExpirer { return c }),
	kessoku.Provide(consumer.New),
	kessoku.Provide(reaper.New),
	kessoku.Provide(worker.New),
)
