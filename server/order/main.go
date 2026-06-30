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
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(start(ctx, ":80"))
}

func start(ctx context.Context, addr string) int {
	if err := run(ctx, addr); err != nil {
		slog.Error("order service terminated", "error", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, addr string) error {
	shutdown, err := otelx.Setup(ctx, "order")
	if err != nil {
		return err
	}
	defer shutdown()

	app, err := di.InitApp(ctx)
	if err != nil {
		return err
	}

	// (ADR-[[202606261212]])
	go func() {
		if err := app.Relay.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("outbox relay terminated", "error", err)
		}
	}()

	engine := httpx.NewEngine()
	api.RegisterHandlers(engine, app.Handler)

	slog.Info("order service listening", "addr", addr)
	return httpx.Serve(ctx, addr, engine)
}
