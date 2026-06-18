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

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) Response {
	t.Helper()
	var body Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return body
}

func TestOnRequestError(t *testing.T) {
	type args struct {
		err error
	}
	type want struct {
		status      int
		code        string
		fullMessage string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系 入力エラーは 400 で文言を透過する",
			args: args{err: errors.New("missing required field")},
			want: want{status: http.StatusBadRequest, code: "bad_request", fullMessage: "missing required field"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newCtx()
			OnRequestError(c, tt.args.err)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			body := decodeBody(t, rec)
			if body.Code != tt.want.code {
				t.Fatalf("code = %q, want %q", body.Code, tt.want.code)
			}
			if body.Message != tt.want.fullMessage {
				t.Fatalf("message = %q, want %q (透過のはず)", body.Message, tt.want.fullMessage)
			}
		})
	}
}

func TestOnHandlerError(t *testing.T) {
	type args struct {
		err error
	}
	type want struct {
		status    int
		code      string
		hideInput string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "異常系 handler エラーは 500 で内部文言を隠す",
			args: args{err: errors.New("connection refused")},
			want: want{status: http.StatusInternalServerError, code: "internal", hideInput: "connection refused"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newCtx()
			OnHandlerError(c, tt.args.err)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			body := decodeBody(t, rec)
			if body.Code != tt.want.code {
				t.Fatalf("code = %q, want %q", body.Code, tt.want.code)
			}
			if body.Message == tt.want.hideInput {
				t.Fatalf("message が内部エラーを露出している: %q", body.Message)
			}
		})
	}
}

func TestOnResponseError(t *testing.T) {
	type args struct {
		err error
	}
	type want struct {
		status    int
		code      string
		hideInput string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "異常系 レスポンス serialize 失敗も 500 で内部文言を隠す",
			args: args{err: errors.New("encode fail")},
			want: want{status: http.StatusInternalServerError, code: "internal", hideInput: "encode fail"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newCtx()
			OnResponseError(c, tt.args.err)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			body := decodeBody(t, rec)
			if body.Code != tt.want.code {
				t.Fatalf("code = %q, want %q", body.Code, tt.want.code)
			}
			if body.Message == tt.want.hideInput {
				t.Fatalf("message が内部エラーを露出している: %q", body.Message)
			}
		})
	}
}
