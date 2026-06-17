package httperror

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
	return c, rec
}

func TestErrorHandlers(t *testing.T) {
	tests := []struct {
		name       string
		handler    func(*gin.Context, error)
		err        error
		wantStatus int
		wantCode   string
		// true なら err.Error() の文言を Message にそのまま透過する (入力エラー)、
		// false なら隠蔽する (内部エラー)。
		exposeErr bool
	}{
		{
			name:       "RequestErrorHandler は 400 と入力エラー文言を返す",
			handler:    RequestErrorHandler,
			err:        errors.New("missing required field"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "bad_request",
			exposeErr:  true,
		},
		{
			name:       "HandlerErrorHandler は 500 で内部詳細を隠蔽する",
			handler:    HandlerErrorHandler,
			err:        errors.New("connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
			exposeErr:  false,
		},
		{
			name:       "ResponseErrorHandler は 500 で内部詳細を隠蔽する",
			handler:    ResponseErrorHandler,
			err:        errors.New("encode fail"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
			exposeErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newCtx()
			tt.handler(c, tt.err)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Fatalf("content-type = %q", ct)
			}
			var body Response
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if tt.exposeErr && body.Message != tt.err.Error() {
				t.Fatalf("message = %q, want %q (透過のはず)", body.Message, tt.err.Error())
			}
			if !tt.exposeErr && body.Message == tt.err.Error() {
				t.Fatalf("message が内部エラーを露出している: %q", body.Message)
			}
		})
	}
}
