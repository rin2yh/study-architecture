package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
	"github.com/rin2yh/study-architecture/server/payment/internal/handler"
	"github.com/rin2yh/study-architecture/server/payment/internal/rdb"
	"github.com/rin2yh/study-architecture/server/payment/internal/stub"
)

func newReadServer(query handler.Query) http.Handler {
	return newServer(handler.New(query, nil))
}

func TestListPayments(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE payment.payments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO payment.payments (order_id, amount_cents, method, status) VALUES ($1, $2, $3, $4)`,
		int64(10), int64(1980), "card", "paid"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewPaymentQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/payments", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []api.Payment{{OrderId: 10, AmountCents: 1980, Method: "card", Status: "paid"}}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Payment{}, "Id", "CreatedAt")); diff != "" {
		t.Fatalf("payments mismatch (-want +got):\n%s", diff)
	}
}

func TestListPaymentsError(t *testing.T) {
	fake := stub.PaymentStub{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newReadServer(fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/payments", nil))

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

func TestGetPayment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	payment := db.PaymentPayment{ID: 1, OrderID: 10, AmountCents: 1980, Method: "card", Status: "paid", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		fake stub.PaymentStub
		path string
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
		{"正常系 決済を返す", args{stub.PaymentStub{Payment: payment}, "/payments/1"}, want{http.StatusOK, ""}},
		{"異常系 未存在は 404 not_found", args{stub.PaymentStub{Err: dberr.ErrNotFound}, "/payments/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.PaymentStub{Err: errors.New("db failure")}, "/payments/1"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newReadServer(tt.args.fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.args.path, nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
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
