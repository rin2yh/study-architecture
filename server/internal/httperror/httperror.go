// Package httperror は oapi-codegen (gin-server) の strict server が呼び出す
// エラー hook の実装を 5 サービスで共有する。
//
// hook は Gin の middleware (gin.HandlerFunc) ではなく、strict server が
// 「リクエストパース失敗 / handler が error を返した / response 化に失敗した」
// それぞれのタイミングで呼ぶ専用コールバック (func(c *gin.Context, err error))。
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

// 入力起因のエラー (binding 失敗等) を 400 で返す。文言は err をそのまま透過してよい。
// StrictGinServerOptions.RequestErrorHandlerFunc に渡す。
func BadRequest(c *gin.Context, err error) {
	slog.WarnContext(c.Request.Context(), "request rejected", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusBadRequest, Response{Code: "bad_request", Message: err.Error()})
}

// 内部エラー (handler の error / response serialize 失敗) を 500 で返す。
// DB 接続情報・スタック等の内部詳細はクライアントへ露出させない。
// StrictGinServerOptions.HandlerErrorFunc と ResponseErrorHandlerFunc の両方に渡す。
func Internal(c *gin.Context, err error) {
	slog.ErrorContext(c.Request.Context(), "internal error", "error", err, "path", c.Request.URL.Path)
	c.AbortWithStatusJSON(http.StatusInternalServerError, Response{Code: "internal", Message: "internal server error"})
}
