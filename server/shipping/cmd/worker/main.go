package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/internal/otelx"
	"github.com/rin2yh/study-architecture/server/shipping/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(start(ctx))
}

func start(ctx context.Context) int {
	if err := run(ctx); err != nil {
		slog.Error("shipping worker terminated", "error", err)
		return 1
	}
	return 0
}

func run(ctx context.Context) error {
	shutdown, err := otelx.Setup(ctx, "shipping-worker")
	if err != nil {
		return err
	}
	defer shutdown()

	cons, err := di.InitConsumer(ctx)
	if err != nil {
		return err
	}

	slog.Info("shipping worker started")
	// context.Canceled は SIGTERM 受信後の正常停止なので error とはみなさない。
	if err := cons.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
