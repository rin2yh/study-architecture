//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/member/internal/handler"
	"github.com/rin2yh/study-architecture/server/member/internal/repository"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(repository.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(repository.NewMemberQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(repository.NewMemberCommand)),
	kessoku.Provide(handler.New),
)
