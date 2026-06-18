package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

const dbEnv = "DATABASE_URL_OPS"

type fakeQuerier struct {
	rows    []db.ProductProduct
	product db.ProductProduct
	err     error
}

func (f fakeQuerier) ListProducts(context.Context) ([]db.ProductProduct, error) {
	return f.rows, f.err
}

func (f fakeQuerier) GetProduct(context.Context, int64) (db.ProductProduct, error) {
	return f.product, f.err
}

func (f fakeQuerier) CreateProduct(context.Context, db.CreateProductParams) (db.ProductProduct, error) {
	return f.product, f.err
}

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
	r := NewRepository(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedProducts(t, pool, tt.seed...)

			got, err := r.ListProducts(context.Background())
			if err != nil {
				t.Fatalf("ListProducts: %v", err)
			}
			if got == nil {
				t.Fatal("ListProducts: want non-nil slice (emit_empty_slices)")
			}
			if diff := cmp.Diff(tt.seed, got,
				cmpopts.IgnoreFields(db.ProductProduct{}, "ID", "CreatedAt"),
				cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListProducts mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryListProductsError(t *testing.T) {
	skip.Short(t)
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ListProducts(ctx); err == nil {
		t.Fatal("ListProducts: want error from canceled context")
	}
}

func TestRepositoryGetProduct(t *testing.T) {
	product := db.ProductProduct{ID: 1, Sku: "SKU-1"}
	other := errors.New("query failed")
	type args struct{ q fakeQuerier }
	type want struct {
		id  int64
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 行を返す", args{fakeQuerier{product: product}}, want{1, nil}},
		{"異常系 no rows は ErrNotFound に正規化", args{fakeQuerier{err: pgx.ErrNoRows}}, want{0, dberr.ErrNotFound}},
		{"異常系 その他エラーは透過", args{fakeQuerier{err: other}}, want{0, other}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).GetProduct(context.Background(), 1)
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetProduct: %v", err)
			}
			if got.ID != tt.want.id {
				t.Fatalf("id = %d, want %d", got.ID, tt.want.id)
			}
		})
	}
}

func TestRepositoryCreateProduct(t *testing.T) {
	created := db.ProductProduct{ID: 10, Sku: "SKU-NEW"}
	other := errors.New("query failed")
	type args struct{ q fakeQuerier }
	type want struct {
		id  int64
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 作成行を返す", args{fakeQuerier{product: created}}, want{10, nil}},
		{"異常系 unique_violation は ErrConflict に正規化", args{fakeQuerier{err: &pgconn.PgError{Code: "23505"}}}, want{0, dberr.ErrConflict}},
		{"異常系 その他エラーは透過", args{fakeQuerier{err: other}}, want{0, other}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).CreateProduct(context.Background(), db.CreateProductParams{})
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateProduct: %v", err)
			}
			if got.ID != tt.want.id {
				t.Fatalf("id = %d, want %d", got.ID, tt.want.id)
			}
		})
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
