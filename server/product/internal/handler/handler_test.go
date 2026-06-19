package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/internal/test/apitest"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
	"github.com/rin2yh/study-architecture/server/product/internal/handler"
	"github.com/rin2yh/study-architecture/server/product/internal/repository"
	"github.com/rin2yh/study-architecture/server/product/internal/stub"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newServer(repo repository.ProductRepository) http.Handler {
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
	newServer(repository.NewRepository(pool)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []api.Product{{Sku: "SKU-DB-1", Name: "DB 商品", PriceCents: 500}}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(api.Product{}, "Id", "CreatedAt")); diff != "" {
		t.Fatalf("products mismatch (-want +got):\n%s", diff)
	}
}

func TestListProductsError(t *testing.T) {
	repo := stub.Repo{Err: errors.New("db failure")}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products", nil))

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
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	product := db.ProductProduct{ID: 1, Sku: "SKU-1", Name: "サンプル商品", PriceCents: 1980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		repo stub.Repo
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
		{"正常系 商品を返す", args{stub.Repo{Product: product}, "/products/1"}, want{http.StatusOK, ""}},
		{"異常系 未存在は 404 not_found", args{stub.Repo{Err: dberr.ErrNotFound}, "/products/99"}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, "/products/1"}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newServer(tt.args.repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.args.path, nil))
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Product
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.Sku != "SKU-1" {
				t.Fatalf("unexpected product: %+v", got)
			}
		})
	}
}

func TestCreateProduct(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	created := db.ProductProduct{ID: 10, Sku: "SKU-NEW", Name: "新規商品", PriceCents: 2980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		repo stub.Repo
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
		{"正常系 商品を作成し 201", args{stub.Repo{Product: created}, `{"sku":"SKU-NEW","name":"新規商品","priceCents":2980}`}, want{http.StatusCreated, ""}},
		{"異常系 sku 欠落は 400 bad_request", args{stub.Repo{}, `{"name":"x","priceCents":100}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 name 欠落は 400 bad_request", args{stub.Repo{}, `{"sku":"SKU-X","priceCents":100}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 priceCents 負値は 422 unprocessable_entity", args{stub.Repo{}, `{"sku":"SKU-X","name":"x","priceCents":-1}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 sku 重複は 409 conflict", args{stub.Repo{Err: dberr.ErrConflict}, `{"sku":"SKU-DUP","name":"重複","priceCents":100}`}, want{http.StatusConflict, "conflict"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, `{"sku":"SKU-X","name":"x","priceCents":100}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Product
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 10 || got.Sku != "SKU-NEW" {
				t.Fatalf("unexpected product: %+v", got)
			}
		})
	}
}

func TestUpdateProduct(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updated := db.ProductProduct{ID: 1, Sku: "SKU-UPD", Name: "更新後商品", PriceCents: 3980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}}
	type args struct {
		repo stub.Repo
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
		{"正常系 商品を更新し 200", args{stub.Repo{Product: updated}, "/products/1", `{"sku":"SKU-UPD","name":"更新後商品","priceCents":3980}`}, want{http.StatusOK, ""}},
		{"異常系 sku 欠落は 400 bad_request", args{stub.Repo{}, "/products/1", `{"name":"x","priceCents":100}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 priceCents 負値は 422 unprocessable_entity", args{stub.Repo{}, "/products/1", `{"sku":"SKU-X","name":"x","priceCents":-1}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 未存在は 404 not_found", args{stub.Repo{Err: dberr.ErrNotFound}, "/products/99", `{"sku":"SKU-X","name":"x","priceCents":100}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 sku 重複は 409 conflict", args{stub.Repo{Err: dberr.ErrConflict}, "/products/1", `{"sku":"SKU-DUP","name":"重複","priceCents":100}`}, want{http.StatusConflict, "conflict"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, "/products/1", `{"sku":"SKU-X","name":"x","priceCents":100}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, tt.args.path, bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.repo).ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want.status, rec.Body.String())
			}
			if tt.want.code != "" {
				apitest.AssertErrorCode(t, rec.Body.Bytes(), tt.want.code)
				return
			}
			var got api.Product
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Id != 1 || got.Sku != "SKU-UPD" {
				t.Fatalf("unexpected product: %+v", got)
			}
		})
	}
}
