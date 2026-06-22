package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/internal/httpx"
	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("product service terminated", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	h, err := di.InitHandler(ctx)
	if err != nil {
		return err
	}

	engine := httpx.NewEngine()
	api.RegisterHandlers(engine, h)

	addr := httpx.ListenAddr()
	slog.Info("product service listening", "addr", addr)
	return httpx.Serve(ctx, addr, engine)
}
