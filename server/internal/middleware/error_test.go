package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newServer は ErrorJSON 配下で、リクエストごとに 1 つだけエラーを積む (または積まない)
// テスト用エンジンを組み立てる。errType が 0 のときは c.Error のデフォルト (Private) のまま。
func newServer(err error, errType gin.ErrorType) http.Handler {
	engine := gin.New()
	engine.Use(middleware.ErrorJSON())
	engine.GET("/", func(c *gin.Context) {
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
		e := c.Error(err)
		if errType != 0 {
			e.Type = errType
		}
	})
	return engine
}

func TestErrorJSON(t *testing.T) {
	type args struct {
		err     error
		errType gin.ErrorType
	}
	type want struct {
		status  int
		code    string // "" のとき body 検証をスキップ (エラー無しケース)
		message string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		// 正常系
		{"正常系 エラー無しは整形せず素通し", args{nil, 0}, want{http.StatusOK, "", ""}},

		// 準正常系 (AppError で表明したドメインセマンティクスをそのまま透過)
		{"準正常系 NotFound は 404 not_found", args{middleware.NotFound("member 99 not found"), 0}, want{http.StatusNotFound, "not_found", "member 99 not found"}},
		{"準正常系 Conflict は 409 conflict", args{middleware.Conflict("email already exists"), 0}, want{http.StatusConflict, "conflict", "email already exists"}},
		{"準正常系 Unprocessable は 422 unprocessable_entity", args{middleware.Unprocessable("price must be positive"), 0}, want{http.StatusUnprocessableEntity, "unprocessable_entity", "price must be positive"}},
		{"準正常系 NewError は任意の status/code を透過", args{middleware.NewError(http.StatusTooManyRequests, "rate_limited", "slow down"), 0}, want{http.StatusTooManyRequests, "rate_limited", "slow down"}},
		{"準正常系 5xx の AppError はサーバ起因としてログ", args{middleware.NewError(http.StatusBadGateway, "upstream", "bad gateway"), 0}, want{http.StatusBadGateway, "upstream", "bad gateway"}},

		// gin のエラー型有無でクライアント入力起因 (4xx) かサーバ起因 (5xx) かを分ける
		{"準正常系 Bind エラーは 400 bad_request で文言透過", args{errors.New("invalid body"), gin.ErrorTypeBind}, want{http.StatusBadRequest, "bad_request", "invalid body"}},
		{"準正常系 Public エラーは 400 bad_request で文言透過", args{errors.New("missing query param"), gin.ErrorTypePublic}, want{http.StatusBadRequest, "bad_request", "missing query param"}},
		{"異常系 通常エラーは 500 internal で内部詳細を隠す", args{errors.New("db connection refused"), 0}, want{http.StatusInternalServerError, "internal", "internal server error"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newServer(tt.args.err, tt.args.errType).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if tt.want.code == "" {
				return
			}
			var body struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if body.Code != tt.want.code {
				t.Fatalf("code = %q, want %q", body.Code, tt.want.code)
			}
			if body.Message != tt.want.message {
				t.Fatalf("message = %q, want %q", body.Message, tt.want.message)
			}
		})
	}
}
