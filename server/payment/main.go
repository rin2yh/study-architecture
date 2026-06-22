package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/internal/httpx"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/di"
)

func main() {
	os.Exit(start())
}

func start() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, ":80"); err != nil {
		slog.Error("payment service terminated", "error", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, addr string) error {
	h, err := di.InitHandler(ctx)
	if err != nil {
		return err
	}

	engine := httpx.NewEngine()
	api.RegisterHandlers(engine, h)

	slog.Info("payment service listening", "addr", addr)
	return httpx.Serve(ctx, addr, engine)
}
