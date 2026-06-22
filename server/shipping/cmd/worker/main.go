package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rin2yh/study-architecture/server/shipping/internal/di"
)

func main() {
	if err := run(); err != nil {
		slog.Error("shipping worker terminated", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cons, err := di.InitConsumer(ctx)
	if err != nil {
		return err
	}

	slog.Info("shipping worker started")
	// silent death させず再起動に委ねる
	if err := cons.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
