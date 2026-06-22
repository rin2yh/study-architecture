// Package httpx は 5 サービスの HTTP サーバ bootstrap (Gin エンジン構築と
// グレースフルシャットダウン) を共通化する。各 main.go が同一処理を重複実装するのを
// 1 箇所に集約する。
package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
)

func NewEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.ErrorJSON())
	return engine
}

func Serve(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Warn("graceful shutdown failed", "error", err)
		}
	}()

	// ErrServerClosed は Shutdown 経由の正常停止なので error とはみなさない。
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
