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
	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/handler"
	"github.com/rin2yh/study-architecture/server/product/internal/rdb"
	"github.com/rin2yh/study-architecture/server/product/internal/stub"
)

func newReadServer(query handler.Query) http.Handler {
	return newServer(handler.New(query, nil))
}

func TestListProducts(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_OPS")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE product.products RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO product.products (sku, name, price_cents) VALUES ($1, $2, $3)`,
		"SKU-DB-1", "DB 商品", 500); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewProductQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cmptest.EqualSlice(t, []api.Product{{Sku: "SKU-DB-1", Name: "DB 商品", PriceCents: 500}}, got, "Id", "CreatedAt")
}

func TestListProductsError(t *testing.T) {
	fake := stub.ProductStub{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newReadServer(fake).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q, want application/json", ct)
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

func TestGetProduct(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_OPS")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE product.products RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO product.products (sku, name, price_cents) VALUES ($1, $2, $3)`,
		"SKU-1", "サンプル商品", 1980); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	newReadServer(rdb.NewProductQuery(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products/1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cmptest.Equal(t, api.Product{Sku: "SKU-1", Name: "サンプル商品", PriceCents: 1980}, got, "Id", "CreatedAt")
}

func TestGetProductError(t *testing.T) {
	type args struct {
		fake stub.ProductStub
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
		{"異常系 未存在は 404 not_found", args{stub.ProductStub{Err: dberr.ErrNotFound}, "/products/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.ProductStub{Err: errors.New("db failure")}, "/products/1"}, want{http.StatusInternalServerError, "internal"}},
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
