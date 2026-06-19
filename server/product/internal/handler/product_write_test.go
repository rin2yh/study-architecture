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
	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
	"github.com/rin2yh/study-architecture/server/product/internal/stub"
)

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
			newServer(tt.args.repo, tt.args.repo).ServeHTTP(rec, req)
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
		{"正常系 商品を更新し 200", args{stub.Repo{Product: updated}, "/products/1", `{"name":"更新後商品","priceCents":3980}`}, want{http.StatusOK, ""}},
		{"異常系 name 欠落は 400 bad_request", args{stub.Repo{}, "/products/1", `{"priceCents":100}`}, want{http.StatusBadRequest, "bad_request"}},
		{"異常系 priceCents 負値は 422 unprocessable_entity", args{stub.Repo{}, "/products/1", `{"name":"x","priceCents":-1}`}, want{http.StatusUnprocessableEntity, "unprocessable_entity"}},
		{"異常系 未存在は 404 not_found", args{stub.Repo{Err: dberr.ErrNotFound}, "/products/99", `{"name":"x","priceCents":100}`}, want{http.StatusNotFound, "not_found"}},
		{"異常系 DB エラーは 500 internal", args{stub.Repo{Err: errors.New("db failure")}, "/products/1", `{"name":"x","priceCents":100}`}, want{http.StatusInternalServerError, "internal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, tt.args.path, bytes.NewReader([]byte(tt.args.body)))
			req.Header.Set("Content-Type", "application/json")
			newServer(tt.args.repo, tt.args.repo).ServeHTTP(rec, req)
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
			if got.Id != 1 || got.Name != "更新後商品" {
				t.Fatalf("unexpected product: %+v", got)
			}
		})
	}
}
