package di

import (
	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/payment/internal/consumer"
	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
)

// App は payment プロセスが回す HTTP ハンドラ・outbox リレー (settled 発行)・consumer (cancelled 受信
// で返金) をまとめる。いずれも同一の DB プールを共有するため (ADR-[[202606261212]])、1 つの DI グラフで
// 束ねて二重接続を避ける。発行 (outbox) と受信 (consumer) はどちらも payment プロセス内に同居する。
type App struct {
	Handler  *handler.Handler
	Relay    *outbox.Relay
	Consumer *consumer.Consumer
}

func NewApp(h *handler.Handler, r *outbox.Relay, c *consumer.Consumer) *App {
	return &App{Handler: h, Relay: r, Consumer: c}
}
