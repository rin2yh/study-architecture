package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/testdb"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても実 SQL が
// schema と噛み合うかは検証できない。DATABASE_URL_OPS が指す実 DB (compose の db-ops /
// CI の service) へ接続して結合テストする。skip 条件は testdb 参照。
const dbEnv = "DATABASE_URL_OPS"

func seedProducts(t *testing.T, pool *pgxpool.Pool, rows ...db.ProductProduct) {
	t.Helper()
	ctx := context.Background()
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

func TestRepositoryListProducts(t *testing.T) {
	type args struct {
		seed []db.ProductProduct
	}
	type want struct {
		skus []string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			args: args{seed: []db.ProductProduct{
				{Sku: "SKU-1", Name: "商品1", PriceCents: 1980},
				{Sku: "SKU-2", Name: "商品2", PriceCents: 2980},
			}},
			want: want{skus: []string{"SKU-1", "SKU-2"}},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			args: args{seed: nil},
			want: want{skus: []string{}},
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedProducts(t, pool, tt.args.seed...)

			got, err := r.ListProducts(context.Background())
			if err != nil {
				t.Fatalf("ListProducts: %v", err)
			}
			if got == nil {
				t.Fatal("ListProducts: want non-nil slice (emit_empty_slices)")
			}
			if len(got) != len(tt.want.skus) {
				t.Fatalf("len = %d, want %d (%+v)", len(got), len(tt.want.skus), got)
			}
			for i, sku := range tt.want.skus {
				if got[i].Sku != sku {
					t.Fatalf("rows[%d].Sku = %q, want %q", i, got[i].Sku, sku)
				}
			}
		})
	}
}

func TestRepositoryListProductsError(t *testing.T) {
	r := NewRepository(testdb.OpenClosed(t, dbEnv))
	if _, err := r.ListProducts(context.Background()); err == nil {
		t.Fatal("ListProducts: want error from closed pool")
	}
}

func TestNewPool(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := NewPool(context.Background()); err == nil {
		t.Fatal("NewPool: want error when DATABASE_URL is empty")
	}

	// ダミー DSN。pgxpool.New は遅延接続なので実際の接続は行われず error にならない。
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	pool, err := NewPool(context.Background())
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	if pool == nil {
		t.Fatal("NewPool: pool is nil")
	}
}

func TestNewRepository(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	pool, err := NewPool(context.Background())
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	if NewRepository(pool) == nil {
		t.Fatal("NewRepository: want non-nil")
	}
}
