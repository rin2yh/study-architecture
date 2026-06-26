package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
	"github.com/rin2yh/study-architecture/server/payment/internal/rdb"
	"github.com/rin2yh/study-architecture/server/payment/internal/stub"
)

func newWriteServer(command handler.Command) http.Handler {
	return newServer(handler.New(nil, command))
}

func TestCreatePayment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_PAYMENT")
	if _, err := pool.Exec(t.Context(), `TRUNCATE payment.payments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader([]byte(`{"orderId":20,"amountCents":2980,"method":"card","status":"paid","idempotencyKey":"key-1"}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewPaymentCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := api.Payment{OrderId: 20, AmountCents: 2980, Method: "card", Status: "paid"}
	assert.DeepEqual(t, want, got, "Id", "CreatedAt")
}

func TestCreatePaymentError(t *testing.T) {
	type args struct {
		fake stub.PaymentStub
		body string
	}
	type want struct {
		status int
		code   string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"準正常系 method 欠落は 400 bad_request", args{stub.PaymentStub{}, `{"orderId":20,"amountCents":2980,"status":"paid","idempotencyKey":"k"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 orderId 欠落は 400 bad_request", args{stub.PaymentStub{}, `{"amountCents":2980,"method":"card","status":"paid","idempotencyKey":"k"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 idempotencyKey 欠落は 400 bad_request", args{stub.PaymentStub{}, `{"orderId":20,"amountCents":2980,"method":"card","status":"paid"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 amountCents 負値は 422 unprocessable_entity", args{stub.PaymentStub{}, `{"orderId":20,"amountCents":-1,"method":"card","status":"paid","idempotencyKey":"k"}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 DB エラーは 500 internal", args{stub.PaymentStub{Err: errors.New("db failure")}, `{"orderId":20,"amountCents":2980,"method":"card","status":"paid","idempotencyKey":"k"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestUpdatePayment(t *testing.T) {
	skip.Short(t)
	type args struct{ body string }
	type want struct {
		status string
		// 確定への更新は同一 tx で outbox の未送信マークが立つ。送出はリレーが後追いする (ADR-[[202606261212]])。
		pending bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 確定 status へ更新すると outbox を未送信でマークする",
			args{`{"status":"paid"}`},
			want{"paid", true},
		},
		{
			"準正常系 非確定 status への更新では outbox をマークしない",
			args{`{"status":"refunded"}`},
			want{"refunded", false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := testdb.Open(t, "DATABASE_URL_PAYMENT")
			ctx := t.Context()
			if _, err := pool.Exec(ctx, `TRUNCATE payment.payments RESTART IDENTITY`); err != nil {
				t.Fatalf("truncate: %v", err)
			}
			if _, err := pool.Exec(ctx,
				`INSERT INTO payment.payments (order_id, amount_cents, method, status) VALUES ($1, $2, $3, $4)`,
				int64(20), int64(2980), "card", "pending"); err != nil {
				t.Fatalf("insert: %v", err)
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, "/payments/1", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(rdb.NewPaymentCommand(pool)).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
			}
			var got api.Payment
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			want := api.Payment{OrderId: 20, AmountCents: 2980, Method: "card", Status: tt.want.status}
			assert.DeepEqual(t, want, got, "Id", "CreatedAt")

			var pending bool
			if err := pool.QueryRow(ctx, `SELECT settled_event_pending FROM payment.payments WHERE id = 1`).Scan(&pending); err != nil {
				t.Fatalf("query pending: %v", err)
			}
			if pending != tt.want.pending {
				t.Fatalf("settled_event_pending = %v, want %v", pending, tt.want.pending)
			}
		})
	}
}

func TestUpdatePaymentError(t *testing.T) {
	type args struct {
		fake stub.PaymentStub
		path string
		body string
	}
	type want struct {
		status int
		code   string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"準正常系 status 欠落は 400 bad_request", args{stub.PaymentStub{}, "/payments/1", `{}`}, want{http.StatusBadRequest, "bad_request"}},
		{"準正常系 未存在は 404 not_found", args{stub.PaymentStub{Err: dberr.ErrNotFound}, "/payments/99", `{"status":"refunded"}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.PaymentStub{Err: errors.New("db failure")}, "/payments/1", `{"status":"refunded"}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, tt.args.path, bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}
