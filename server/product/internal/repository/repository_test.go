package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても
// 実 SQL が schema と噛み合うかは検証できない。ここでは DATABASE_URL_OPS が指す
// 実 DB (compose の db-ops / CI の service) へ接続して結合テストする。
//
// ビルドタグは使わず、-short 実行時 (per-service の単体ジョブ) と DSN 未設定時に
// skip することで、DB が無い環境でも `go test ./...` が通るようにしている。
func testDSN(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skip integration test in -short mode")
	}
	dsn := os.Getenv("DATABASE_URL_OPS")
	if dsn == "" {
		t.Skip("skip integration test: DATABASE_URL_OPS is not set")
	}
	return dsn
}

func openTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), testDSN(t))
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// seedProducts は table を空にしてから rows を id 昇順 (= 挿入順) で入れ直す。
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

	pool := openTestDB(t)
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

// 異常系: 接続不能 (閉じた pool) でクエリがエラーを伝播することを確認する。
func TestRepositoryListProductsError(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), testDSN(t))
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	pool.Close()

	r := NewRepository(pool)
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
