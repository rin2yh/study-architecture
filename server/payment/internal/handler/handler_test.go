package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
	"github.com/rin2yh/study-architecture/server/payment/internal/repository"
	"github.com/rin2yh/study-architecture/server/payment/internal/stub"
)

// assertErrorCode は共通エラー JSON ({code,message}) の code を検証する。
func assertErrorCode(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if e.Code != wantCode {
		t.Fatalf("code = %q, want %q", e.Code, wantCode)
	}
}

func init() {
	gin.SetMode(gin.TestMode)
}

func newServer(repo repository.PaymentRepository) http.Handler {
	engine := gin.New()
	engine.Use(middleware.ErrorJSON())
	api.RegisterHandlers(engine, handler.New(repo))
	return engine
}

func TestGetHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(stub.Repo{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body api.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q, want ok", body.Status)
	}
}

func TestListPayments(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := stub.Repo{Payments: []db.PaymentPayment{
		{ID: 1, OrderID: 10, AmountCents: 1980, Method: "card", Status: "paid", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/payments", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].OrderId != 10 || got[0].AmountCents != 1980 || got[0].Method != "card" || got[0].Status != "paid" || !got[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected payment: %+v", got[0])
	}
}

func TestListPaymentsError(t *testing.T) {
	repo := stub.Repo{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/payments", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != "internal" {
		t.Fatalf("code = %q, want internal", body.Code)
	}
	if body.Message == "db failure" {
		t.Fatalf("message must not expose internal error: %q", body.Message)
	}
}

// handler は presentation 層なので、stub だけでなく実 DB を通した経路でも検証する
// (skip 条件は testdb 参照)。
func TestListPaymentsWithDB(t *testing.T) {
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE payment.payments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO payment.payments (order_id, amount_cents, method, status) VALUES ($1, $2, $3, $4)`,
		int64(10), int64(1980), "card", "paid"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newServer(repository.NewRepository(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/payments", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].OrderId != 10 || got[0].Method != "card" || got[0].AmountCents != 1980 {
		t.Fatalf("unexpected payment: %+v", got)
	}
}

func TestGetPayment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	payment := db.PaymentPayment{ID: 1, OrderID: 10, AmountCents: 1980, Method: "card", Status: "paid", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		repo stub.Repo
		path string
	}
	type want struct {
		status int
		code   string // "" のとき成功 (Payment body を検証)
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 決済を返す", args{stub.Repo{Payment: payment}, "/payments/1"}, want{http.StatusOK, ""}},
		{"異常系 未存在は 404 not_found", args{stub.Repo{Err: dberr.ErrNotFound}, "/payments/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, "/payments/1"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newServer(tt.args.repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.args.path, nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if tt.want.code != "" {
				assertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Payment
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.OrderId != 10 {
				t.Fatalf("unexpected payment: %+v", got)
			}
		})
	}
}

func TestCreatePayment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	created := db.PaymentPayment{ID: 10, OrderID: 20, AmountCents: 2980, Method: "card", Status: "paid", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		repo stub.Repo
		body string
	}
	type want struct {
		status int
		code   string // "" のとき成功 (201 + Payment body を検証)
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 決済を作成し 201", args{stub.Repo{Payment: created}, `{"orderId":20,"amountCents":2980,"method":"card","status":"paid"}`}, want{http.StatusCreated, ""}},
		{"異常系 method 欠落は 400 bad_request", args{stub.Repo{}, `{"orderId":20,"amountCents":2980,"status":"paid"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 orderId 欠落は 400 bad_request", args{stub.Repo{}, `{"amountCents":2980,"method":"card","status":"paid"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 amountCents 負値は 422 unprocessable_entity", args{stub.Repo{}, `{"orderId":20,"amountCents":-1,"method":"card","status":"paid"}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, `{"orderId":20,"amountCents":2980,"method":"card","status":"paid"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				assertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Payment
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 10 || got.OrderId != 20 {
				t.Fatalf("unexpected payment: %+v", got)
			}
		})
	}
}
