package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
	"github.com/rin2yh/study-architecture/server/payment/internal/stub"
)

func TestCreatePayment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	created := db.PaymentPayment{ID: 10, OrderID: 20, AmountCents: 2980, Method: "card", Status: "paid", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
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
		{"正常系 決済を作成し 201", args{stub.PaymentStub{Payment: created}, `{"orderId":20,"amountCents":2980,"method":"card","status":"paid"}`}, want{http.StatusCreated, ""}},
		{"異常系 method 欠落は 400 bad_request", args{stub.PaymentStub{}, `{"orderId":20,"amountCents":2980,"status":"paid"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 orderId 欠落は 400 bad_request", args{stub.PaymentStub{}, `{"amountCents":2980,"method":"card","status":"paid"}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 amountCents 負値は 422 unprocessable_entity", args{stub.PaymentStub{}, `{"orderId":20,"amountCents":-1,"method":"card","status":"paid"}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 DB エラーは 500 internal", args{stub.PaymentStub{Err: errors.New("db failure")}, `{"orderId":20,"amountCents":2980,"method":"card","status":"paid"}`}, want{http.StatusInternalServerError, "internal"}},
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
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
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

func TestUpdatePayment(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := db.PaymentPayment{ID: 1, OrderID: 20, AmountCents: 3980, Method: "bank", Status: "refunded", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
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
		{"正常系 決済を更新し 200", args{stub.PaymentStub{Payment: updated}, "/payments/1", `{"status":"refunded"}`}, want{http.StatusOK, ""}},
		{"異常系 status 欠落は 400 bad_request", args{stub.PaymentStub{}, "/payments/1", `{}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在は 404 not_found", args{stub.PaymentStub{Err: dberr.ErrNotFound}, "/payments/99", `{"status":"refunded"}`}, want{http.StatusNotFound, "not_found"}},
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
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Payment
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.Status != "refunded" {
				t.Fatalf("unexpected payment: %+v", got)
			}
		})
	}
}
