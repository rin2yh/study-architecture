// Package middleware は 5 サービス共通の Gin ミドルウェアを提供する。
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// handler が c.Error(err) で積んだ最後のエラーを JSON に整形する。
// gin.ErrorTypeBind / ErrorTypePublic を 400 (文言透過)、それ以外を 500 (内部詳細を隠す) とみなす。
func ErrorJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		last := c.Errors.Last()
		if last.Type&(gin.ErrorTypeBind|gin.ErrorTypePublic) != 0 {
			slog.WarnContext(c.Request.Context(), "request rejected", "error", last.Err, "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{Code: "bad_request", Message: last.Err.Error()})
			return
		}
		slog.ErrorContext(c.Request.Context(), "internal error", "error", last.Err, "path", c.Request.URL.Path)
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse{Code: "internal", Message: "internal server error"})
	}
}
