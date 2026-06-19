//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/shipping/internal/handler"
	"github.com/rin2yh/study-architecture/server/shipping/internal/repository"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(repository.NewShipmentQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(repository.NewShipmentCommand)),
	kessoku.Provide(handler.New),
)
