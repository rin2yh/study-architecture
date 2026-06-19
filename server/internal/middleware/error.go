// Package middleware は 5 サービス共通の Gin ミドルウェアを提供する。
package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// errorResponse は全サービス共通のエラー JSON。各サービスの OpenAPI が持つ
// Error スキーマ (code / message) と shape を一致させる。
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// AppError は handler が HTTP ステータスを明示的に表明するための型付きエラー。
// c.Error(err) で積むと ErrorJSON が Status / Code / Message を尊重して整形する。
//
// 従来 ErrorJSON は 400 (Bind/Public) と 500 (それ以外) の 2 値しか返せなかった。
// 404 / 409 / 422 といったドメインのセマンティクスを表現する手段をここに集約し、
// status とエラーコードのマッピングを 1 箇所で管理する。
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Message }

// NewError は任意の status / code / message で AppError を作る。
func NewError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// NotFound は 404 (リソースが存在しない) を表す。
func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

// Conflict は 409 (状態の競合・一意制約違反など) を表す。
func Conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "conflict", Message: message}
}

// Unprocessable は 422 (構文は妥当だが意味的に処理できない入力) を表す。
func Unprocessable(message string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: "unprocessable_entity", Message: message}
}

// BadGateway は 502 (下流サービス呼び出しの失敗) を表す。横断連携を持つ handler が
// 上流の障害を表明するのに使う。
func BadGateway(message string) *AppError {
	return &AppError{Status: http.StatusBadGateway, Code: "bad_gateway", Message: message}
}

// ErrorJSON は handler が c.Error(err) で積んだ最後のエラーを共通フォーマットの
// JSON に整形する。マッピング規則:
//
//   - *AppError                      → 表明された Status / Code / Message (文言透過)
//   - gin.ErrorTypeBind / Public     → 400 bad_request (文言透過)
//   - それ以外                        → 500 internal (内部詳細を隠す)
func ErrorJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		last := c.Errors.Last()

		var appErr *AppError
		switch {
		case errors.As(last.Err, &appErr):
			logRejected(c, appErr.Status, last.Err)
			c.AbortWithStatusJSON(appErr.Status, errorResponse{Code: appErr.Code, Message: appErr.Message})
		case last.Type&(gin.ErrorTypeBind|gin.ErrorTypePublic) != 0:
			logRejected(c, http.StatusBadRequest, last.Err)
			c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse{Code: "bad_request", Message: last.Err.Error()})
		default:
			slog.ErrorContext(c.Request.Context(), "internal error", "error", last.Err, "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse{Code: "internal", Message: "internal server error"})
		}
	}
}

// logRejected は status に応じてログレベルを選ぶ。5xx (サーバ起因) は Error、
// 4xx (クライアント起因) は Warn で記録する。
func logRejected(c *gin.Context, status int, err error) {
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "request failed", "status", status, "error", err, "path", c.Request.URL.Path)
		return
	}
	slog.WarnContext(c.Request.Context(), "request rejected", "status", status, "error", err, "path", c.Request.URL.Path)
}
