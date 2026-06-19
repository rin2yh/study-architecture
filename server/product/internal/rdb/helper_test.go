package rdb

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

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
