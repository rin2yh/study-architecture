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
	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/handler"
	"github.com/rin2yh/study-architecture/server/product/internal/rdb"
	"github.com/rin2yh/study-architecture/server/product/internal/stub"
)

func newWriteServer(command handler.Command) http.Handler {
	return newServer(handler.New(nil, command))
}

func TestCreateProduct(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_OPS")
	if _, err := pool.Exec(t.Context(), `TRUNCATE product.products RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(`{"sku":"SKU-NEW","name":"新規商品","priceCents":2980}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewProductCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assert.DeepEqual(t, api.Product{Sku: "SKU-NEW", Name: "新規商品", PriceCents: 2980}, got, "Id", "CreatedAt")
}

func TestCreateProductError(t *testing.T) {
	type args struct {
		fake stub.ProductStub
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
		{"異常系 sku 欠落は 400 bad_request", args{stub.ProductStub{}, `{"name":"x","priceCents":100}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 name 欠落は 400 bad_request", args{stub.ProductStub{}, `{"sku":"SKU-X","priceCents":100}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 priceCents 負値は 422 unprocessable_entity", args{stub.ProductStub{}, `{"sku":"SKU-X","name":"x","priceCents":-1}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 sku 重複は 409 conflict", args{stub.ProductStub{Err: dberr.ErrConflict}, `{"sku":"SKU-DUP","name":"重複","priceCents":100}`}, want{http.StatusConflict, "conflict"}},
		{"異常系 DB エラーは 500 internal", args{stub.ProductStub{Err: errors.New("db failure")}, `{"sku":"SKU-X","name":"x","priceCents":100}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newWriteServer(tt.args.fake).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			assert.ErrorCode(t, rec.Body.Bytes(), tt.want.code)
		})
	}
}

func TestUpdateProduct(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, "DATABASE_URL_OPS")
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE product.products RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO product.products (sku, name, price_cents) VALUES ($1, $2, $3)`,
		"SKU-UPD", "更新前商品", 1980); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/products/1", bytes.NewReader([]byte(`{"name":"更新後商品","priceCents":3980}`)))
	req.Header.Set("Content-Type", "application/json")
	newWriteServer(rdb.NewProductCommand(pool)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var got api.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assert.DeepEqual(t, api.Product{Sku: "SKU-UPD", Name: "更新後商品", PriceCents: 3980}, got, "Id", "CreatedAt")
}

func TestUpdateProductError(t *testing.T) {
	type args struct {
		fake stub.ProductStub
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
		{"異常系 name 欠落は 400 bad_request", args{stub.ProductStub{}, "/products/1", `{"priceCents":100}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 priceCents 負値は 422 unprocessable_entity", args{stub.ProductStub{}, "/products/1", `{"name":"x","priceCents":-1}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 未存在は 404 not_found", args{stub.ProductStub{Err: dberr.ErrNotFound}, "/products/99", `{"name":"x","priceCents":100}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.ProductStub{Err: errors.New("db failure")}, "/products/1", `{"name":"x","priceCents":100}`}, want{http.StatusInternalServerError, "internal"}},
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
