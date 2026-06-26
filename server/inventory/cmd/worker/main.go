package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/internal/otelx"
	"github.com/rin2yh/study-architecture/server/inventory/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(start(ctx))
}

func start(ctx context.Context) int {
	if err := run(ctx); err != nil {
		slog.Error("inventory worker terminated", "error", err)
		return 1
	}
	return 0
}

func run(ctx context.Context) error {
	shutdown, err := otelx.Setup(ctx, "inventory-worker")
	if err != nil {
		return err
	}
	defer shutdown()

	w, err := di.InitWorker(ctx)
	if err != nil {
		return err
	}

	slog.Info("inventory worker started")
	if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
