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
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/stub"
)

func newWriteServer(command handler.Command) http.Handler {
	return newServer(handler.New(nil, command, nil, nil))
}

func TestCreateOrder(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	created := db.OrderOrder{ID: 10, MemberID: 20, Status: "pending", TotalCents: 1980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		fake stub.OrderStub
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
		{"正常系 注文を作成し 201", args{stub.OrderStub{Order: created}, `{"memberId":20,"status":"pending","totalCents":1980}`}, want{http.StatusCreated, ""}},
		{"異常系 status 欠落は 400 bad_request", args{stub.OrderStub{}, `{"memberId":20,"totalCents":1980}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 memberId 欠落は 400 bad_request", args{stub.OrderStub{}, `{"status":"pending","totalCents":1980}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 totalCents 負値は 422 unprocessable_entity", args{stub.OrderStub{}, `{"memberId":20,"status":"pending","totalCents":-1}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 DB エラーは 500 internal", args{stub.OrderStub{Err: errors.New("db failure")}, `{"memberId":20,"status":"pending","totalCents":1980}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Order
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 10 || got.MemberId != 20 {
				t.Fatalf("unexpected order: %+v", got)
			}
		})
	}
}

func TestUpdateOrder(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := db.OrderOrder{ID: 1, MemberID: 20, Status: "paid", TotalCents: 4980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		fake stub.OrderStub
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
		{"正常系 注文を更新し 200", args{stub.OrderStub{Order: updated}, "/orders/1", `{"status":"paid"}`}, want{http.StatusOK, ""}},
		{"異常系 status 欠落は 400 bad_request", args{stub.OrderStub{}, "/orders/1", `{}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 未存在は 404 not_found", args{stub.OrderStub{Err: dberr.ErrNotFound}, "/orders/99", `{"status":"paid"}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.OrderStub{Err: errors.New("db failure")}, "/orders/1", `{"status":"paid"}`}, want{http.StatusInternalServerError, "internal"}},
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
			var got api.Order
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.Status != "paid" {
				t.Fatalf("unexpected order: %+v", got)
			}
		})
	}
}
