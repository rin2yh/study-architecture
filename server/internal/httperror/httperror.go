// Package httperror は 5 サービス共通のエラーレスポンス形式と Gin 用ハンドラを提供する。
//
// oapi-codegen (gin-server) の StrictGinServerOptions に渡し、リクエストパース失敗・
// ハンドラエラー・レスポンスシリアライズ失敗の 3 系統を JSON で統一して返す。
// 内部実装の詳細 (DB 接続エラー等) はクライアントへ露出させず、サーバログにのみ残す。
package httperror

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"log/slog"
)

// Response はサービス共通のエラー JSON 形式。
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RequestErrorHandler は path/query/body のバインディング失敗を 400 として JSON で返す。
// 入力に依存する文言なのでメッセージは露出させてよい。
func RequestErrorHandler(c *gin.Context, err error) {
	slog.WarnContext(c.Request.Context(), "request rejected", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{Code: "bad_request", Message: err.Error()})
}

// HandlerErrorHandler は handler が返したエラー (主に DB 失敗) を 500 として JSON で返す。
// 内部詳細はクライアントへ露出させない。
func HandlerErrorHandler(c *gin.Context, err error) {
	slog.ErrorContext(c.Request.Context(), "handler error", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Code: "internal", Message: "internal server error"})
}

// ResponseErrorHandler はレスポンスのシリアライズ失敗を 500 として JSON で返す。
// 内部詳細はクライアントへ露出させない。
func ResponseErrorHandler(c *gin.Context, err error) {
	slog.ErrorContext(c.Request.Context(), "response serialization error", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Code: "internal", Message: "internal server error"})
}
