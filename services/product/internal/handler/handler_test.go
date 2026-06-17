package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-service-base-architecture/services/product/api"
	"github.com/rin2yh/study-service-base-architecture/services/product/internal/db"
)

// stubRepo は DB なしで handler を検証するための ProductRepository スタブ。
type stubRepo struct {
	products []db.ProductProduct
	err      error
}

func (s stubRepo) ListProducts(context.Context) ([]db.ProductProduct, error) {
	return s.products, s.err
}

func newServer(repo stubRepo) http.Handler {
	si := api.NewStrictHandler(New(repo), nil)
	return api.HandlerFromMux(si, http.NewServeMux())
}

func TestGetHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newServer(stubRepo{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

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
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := stubRepo{products: []db.ProductProduct{
		{ID: 1, Sku: "SKU-1", Name: "サンプル商品", PriceCents: 1980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}

	rec := httptest.NewRecorder()
	newServer(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []api.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Sku != "SKU-1" || got[0].PriceCents != 1980 || !got[0].CreatedAt.Equal(now) {
		t.Fatalf("unexpected product: %+v", got[0])
	}
}
