// Package httperror は 5 サービス共通のエラーレスポンス形式と Gin 用ハンドラを提供する。
// oapi-codegen (gin-server) の StrictGinServerOptions に渡して使う。
package httperror

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"log/slog"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// 入力起因のエラーなので文言を透過してよい。
func OnRequestError(c *gin.Context, err error) {
	slog.WarnContext(c.Request.Context(), "request rejected", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{Code: "bad_request", Message: err.Error()})
}

// 内部詳細をクライアントへ露出させない (DB 接続情報・スタックを漏らさない)。
func OnHandlerError(c *gin.Context, err error) {
	slog.ErrorContext(c.Request.Context(), "handler error", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Code: "internal", Message: "internal server error"})
}

// 内部詳細をクライアントへ露出させない。
func OnResponseError(c *gin.Context, err error) {
	slog.ErrorContext(c.Request.Context(), "response serialization error", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Code: "internal", Message: "internal server error"})
}
