package di

import (
	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
)

// App は order プロセスが回す HTTP ハンドラと outbox リレーをまとめる。両者は同一の DB プールを
// 共有するため (ADR-[[202606261212]])、1 つの DI グラフで束ねて二重接続を避ける。
type App struct {
	Handler *handler.Handler
	Relay   *outbox.Relay
}

func NewApp(h *handler.Handler, r *outbox.Relay) *App {
	return &App{Handler: h, Relay: r}
}
