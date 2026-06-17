package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rin2yh/study-service-base-architecture/services/payment/api"
	"github.com/rin2yh/study-service-base-architecture/services/payment/internal/di"
)

func main() {
	if err := run(); err != nil {
		slog.Error("payment service terminated", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// kessoku 生成の injector。Async(NewPool) を含むため ctx を受け取る。
	h, err := di.InitHandler(ctx)
	if err != nil {
		return err
	}

	si := api.NewStrictHandler(h, nil)
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              httpAddr(),
		Handler:           api.HandlerFromMux(si, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("payment service listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func httpAddr() string {
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		return addr
	}
	return ":8080"
}
