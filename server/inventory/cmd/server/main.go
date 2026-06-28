package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/internal/httpx"
	"github.com/rin2yh/study-architecture/server/internal/otelx"
	"github.com/rin2yh/study-architecture/server/inventory/api"
	"github.com/rin2yh/study-architecture/server/inventory/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(start(ctx, ":80"))
}

func start(ctx context.Context, addr string) int {
	if err := run(ctx, addr); err != nil {
		slog.Error("inventory service terminated", "error", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, addr string) error {
	shutdown, err := otelx.Setup(ctx, "inventory")
	if err != nil {
		return err
	}
	defer shutdown()

	h, err := di.InitHandler(ctx)
	if err != nil {
		return err
	}

	engine := httpx.NewEngine()
	api.RegisterHandlers(engine, h)

	slog.Info("inventory service listening", "addr", addr)
	return httpx.Serve(ctx, addr, engine)
}
