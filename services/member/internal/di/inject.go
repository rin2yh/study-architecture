//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-service-base-architecture/services/member/internal/handler"
	"github.com/rin2yh/study-service-base-architecture/services/member/internal/repository"
)

// InitHandler の injector 定義。
//
// pgxpool -> repository -> handler を 1 本に組み立てる。
// Async でラップした NewPool は I/O を伴うため並列初期化対象になり、
// 生成関数は InitHandler(ctx) (*handler.Handler, error) となる（kessoku 生成: inject_band.go）。
var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[repository.MemberRepository](kessoku.Provide(repository.NewRepository)),
	kessoku.Provide(handler.New),
)
