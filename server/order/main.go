package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/internal/httpx"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, ":80"); err != nil {
		slog.Error("order service terminated", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, addr string) error {
	h, err := di.InitHandler(ctx)
	if err != nil {
		return err
	}

	engine := httpx.NewEngine()
	api.RegisterHandlers(engine, h)

	slog.Info("order service listening", "addr", addr)
	return httpx.Serve(ctx, addr, engine)
}
