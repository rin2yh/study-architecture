package rdb

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/cmptest"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

const dbEnv = "DATABASE_URL_OPS"

func seedProducts(t *testing.T, pool *pgxpool.Pool, rows ...db.ProductProduct) {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE product.products RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO product.products (sku, name, price_cents) VALUES ($1, $2, $3)`,
			r.Sku, r.Name, r.PriceCents); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func TestListProducts(t *testing.T) {
	skip.Short(t)
	tests := []struct {
		name string
		seed []db.ProductProduct
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			seed: []db.ProductProduct{
				{Sku: "SKU-1", Name: "商品1", PriceCents: 1980},
				{Sku: "SKU-2", Name: "商品2", PriceCents: 2980},
			},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			seed: nil,
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewProductQuery(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedProducts(t, pool, tt.seed...)

			got, err := r.ListProducts(t.Context())
			if err != nil {
				t.Fatalf("ListProducts: %v", err)
			}
			if got == nil {
				t.Fatal("ListProducts: want non-nil slice (emit_empty_slices)")
			}
			cmptest.EqualSlice(t, tt.seed, got, "ID", "CreatedAt")
		})
	}
}

func TestListProductsError(t *testing.T) {
	skip.Short(t)
	r := NewProductQuery(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListProducts(ctx); err == nil {
		t.Fatal("ListProducts: want error from canceled context")
	}
}

func TestGetProduct(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewProductQuery(pool)
	seedProducts(t, pool, db.ProductProduct{Sku: "SKU-1", Name: "商品1", PriceCents: 1980})

	t.Run("正常系 既存 id の行を返す", func(t *testing.T) {
		got, err := r.GetProduct(t.Context(), 1)
		if err != nil {
			t.Fatalf("GetProduct: %v", err)
		}
		if got.Sku != "SKU-1" {
			t.Fatalf("sku = %q, want SKU-1", got.Sku)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetProduct(t.Context(), 9999); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
