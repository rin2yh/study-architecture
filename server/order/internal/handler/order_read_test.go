package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/handler"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
	"github.com/rin2yh/study-architecture/server/order/internal/stub"
)

func newReadServer(query handler.Query) http.Handler {
	return newServer(handler.New(query, nil, nil, nil))
}

func TestListOrders(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".order_items, "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO "order".orders (member_id, status, total_cents) VALUES ($1, $2, $3)`,
		int64(10), "paid", int64(5000)); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewOrderQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assert.EqualSlice(t, []api.Order{{MemberId: 10, Status: "paid", TotalCents: 5000}}, got, "Id", "CreatedAt")
}

func TestListOrdersError(t *testing.T) {
	rec := httptest.NewRecorder()
	newReadServer(stub.OrderStub{Err: errors.New("db failure")}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders", nil))

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

func TestGetOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".order_items, "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO "order".orders (member_id, status, total_cents) VALUES ($1, $2, $3)`,
		int64(10), "paid", int64(2500)); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO "order".order_items (order_id, product_id, product_name, unit_price_cents, quantity)
		 VALUES (1, 100, 'Widget', 500, 2), (1, 200, 'Gadget', 1500, 1)`); err != nil {
		t.Fatalf("insert items: %v", err)
	}

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewOrderQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders/1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := api.Order{MemberId: 10, Status: "paid", TotalCents: 2500, Items: &[]api.OrderItem{
		{ProductId: 100, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2},
		{ProductId: 200, ProductName: "Gadget", UnitPriceCents: 1500, Quantity: 1},
	}}
	assert.Equal(t, want, got, "Id", "CreatedAt")
}

func TestGetOrderError(t *testing.T) {
	type args struct {
		fake stub.OrderStub
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
		{"異常系 未存在は 404 not_found", args{stub.OrderStub{Err: dberr.ErrNotFound}, "/orders/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.OrderStub{Err: errors.New("db failure")}, "/orders/1"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newReadServer(tt.args.fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.args.path, nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}
