// Package httpx は 5 サービスの HTTP サーバ bootstrap (Gin エンジン構築と
// グレースフルシャットダウン) を共通化する。各 main.go が同一処理を重複実装するのを
// 1 箇所に集約する。
package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
)

func NewEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	// マスキングは既定を崩さない (ADR-[[202606250141]])。
	engine.Use(otelgin.Middleware(os.Getenv("OTEL_SERVICE_NAME")))
	engine.Use(gin.Recovery())
	engine.Use(middleware.ErrorJSON())
	return engine
}

// 素の http.Client では traceparent が伝播せずトレースが切れるため、サービス間呼び出しは
// otelhttp で計装したこの共有クライアントを WithHTTPClient で共用する。
func NewHTTPClient() *http.Client {
	return &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
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
