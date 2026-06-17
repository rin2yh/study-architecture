//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
	"github.com/rin2yh/study-architecture/server/payment/internal/repository"
)

// Async でラップした NewPool は I/O を伴うため並列初期化対象になり、
// 生成関数は InitHandler(ctx) (*handler.Handler, error) となる（kessoku 生成: inject_band.go）。
var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[repository.PaymentRepository](kessoku.Provide(repository.NewRepository)),
	kessoku.Provide(handler.New),
)
