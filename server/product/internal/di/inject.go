//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/product/internal/handler"
	"github.com/rin2yh/study-architecture/server/product/internal/repository"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(repository.NewProductQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(repository.NewProductCommand)),
	kessoku.Provide(handler.New),
)
