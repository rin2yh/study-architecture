//go:generate go tool kessoku $GOFILE

package di

import (
	"github.com/mazrean/kessoku"

	"github.com/rin2yh/study-architecture/server/member/internal/handler"
	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
)

var _ = kessoku.Inject[*handler.Handler](
	"InitHandler",
	kessoku.Async(kessoku.Provide(rdb.NewPool)),
	kessoku.Bind[handler.Query](kessoku.Provide(rdb.NewMemberQuery)),
	kessoku.Bind[handler.Command](kessoku.Provide(rdb.NewMemberCommand)),
	kessoku.Provide(handler.New),
)
