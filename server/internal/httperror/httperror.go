// Package httperror は 5 サービス共通のエラーレスポンス形式とハンドラを提供する。
//
// oapi-codegen の StrictHandler に渡す Request/Response エラーハンドラとして使うことで、
// サービス境界で text/plain ではなく JSON を返し、レスポンス形式を統一する。
// 内部実装の詳細 (DB 接続エラー等) はクライアントへ露出させず、サーバログにのみ残す。
package httperror

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Response はサービス共通のエラー JSON 形式。
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON は status と code/message から JSON エラーを書き出す。
func JSON(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Code: code, Message: message})
}

// RequestErrorHandler は oapi-codegen の RequestErrorHandlerFunc 用。
// path/query/body のバインディング失敗を 400 として JSON で返す。
// 入力に依存する文言なのでメッセージは露出させてよい。
func RequestErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.WarnContext(r.Context(), "request rejected", "error", err, "path", r.URL.Path)
	JSON(w, http.StatusBadRequest, "bad_request", err.Error())
}

// ResponseErrorHandler は oapi-codegen の ResponseErrorHandlerFunc 用。
// handler が返したエラー (主に DB 失敗) を 500 として JSON で返す。
// 内部詳細はクライアントへ露出させない。
func ResponseErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "handler error", "error", err, "path", r.URL.Path)
	JSON(w, http.StatusInternalServerError, "internal", "internal server error")
}
