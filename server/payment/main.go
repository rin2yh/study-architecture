package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/internal/httpx"
	"github.com/rin2yh/study-architecture/server/internal/otelx"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(start(ctx, ":80"))
}

func start(ctx context.Context, addr string) int {
	if err := run(ctx, addr); err != nil {
		slog.Error("payment service terminated", "error", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, addr string) error {
	shutdown, err := otelx.Setup(ctx, "payment")
	if err != nil {
		return err
	}
	defer shutdown()

	app, err := di.InitApp(ctx)
	if err != nil {
		return err
	}

	// 決済確定イベントの送出は outbox リレーをプロセス内で回して後追いする (ADR-[[202606261212]])。
	go func() {
		if err := app.Relay.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("outbox relay terminated", "error", err)
		}
	}()

	// 返金は逆流イベント (order.cancelled) 受信なので発行リレーと同じくプロセス内に同居させる
	// (ADR-[[202606261702]])。
	go func() {
		if err := app.Consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("payment consumer terminated", "error", err)
		}
	}()

	engine := httpx.NewEngine()
	api.RegisterHandlers(engine, app.Handler)

	slog.Info("payment service listening", "addr", addr)
	return httpx.Serve(ctx, addr, engine)
}
