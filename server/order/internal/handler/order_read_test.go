package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	"github.com/rin2yh/study-architecture/server/internal/test/cmptest"
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
	cmptest.EqualSlice(t, []api.Order{{MemberId: 10, Status: "paid", TotalCents: 5000}}, got, "Id", "CreatedAt")
}

func TestListOrdersFilter(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_CUSTOMER")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".order_items, "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO "order".orders (member_id, status, total_cents) VALUES (10,'paid',5000),(20,'paid',3000),(10,'pending',1980)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	server := newReadServer(rdb.NewOrderQuery(pool))

	listOrders := func(t *testing.T, header string) []api.Order {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		if header != "" {
			req.Header.Set("X-Member-Id", header)
		}
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
		}
		var got []api.Order
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return got
	}

	t.Run("正常系 X-Member-Id 付きは本人分だけ返す", func(t *testing.T) {
		cmptest.EqualSlice(t, []api.Order{
			{MemberId: 10, Status: "paid", TotalCents: 5000},
			{MemberId: 10, Status: "pending", TotalCents: 1980},
		}, listOrders(t, "10"), "Id", "CreatedAt")
	})
	t.Run("準正常系 ヘッダ無しは全件返す", func(t *testing.T) {
		cmptest.EqualSlice(t, []api.Order{
			{MemberId: 10, Status: "paid", TotalCents: 5000},
			{MemberId: 20, Status: "paid", TotalCents: 3000},
			{MemberId: 10, Status: "pending", TotalCents: 1980},
		}, listOrders(t, ""), "Id", "CreatedAt")
	})
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
	cmptest.Equal(t, want, got, "Id", "CreatedAt")
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
			apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}
