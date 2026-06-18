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

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/di"
	"github.com/rin2yh/study-architecture/server/internal/httperror"
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

	h, err := di.InitHandler(ctx)
	if err != nil {
		return err
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	si := api.NewStrictHandlerWithOptions(h, nil, api.StrictGinServerOptions{
		RequestErrorHandlerFunc:  httperror.OnRequestError,
		HandlerErrorFunc:         httperror.OnHandlerError,
		ResponseErrorHandlerFunc: httperror.OnResponseError,
	})
	api.RegisterHandlers(engine, si)
	srv := &http.Server{
		Addr:              ":80",
		Handler:           engine,
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
