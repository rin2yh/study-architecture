package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

// fakeQuerier は db.Querier を満たし、Repository.q へ差し替えて DB なしで検証する。
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

func TestRepositoryListProducts(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r := &Repository{q: fakeQuerier{rows: []db.ProductProduct{
		{ID: 1, Sku: "SKU-1", Name: "サンプル商品", PriceCents: 1980, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}}

	got, err := r.ListProducts(context.Background())
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(got) != 1 || got[0].Sku != "SKU-1" {
		t.Fatalf("unexpected rows: %+v", got)
	}
}

func TestRepositoryListProductsError(t *testing.T) {
	want := errors.New("query failed")
	r := &Repository{q: fakeQuerier{err: want}}

	if _, err := r.ListProducts(context.Background()); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestRepositoryGetProduct(t *testing.T) {
	product := db.ProductProduct{ID: 1, Sku: "SKU-1"}
	other := errors.New("query failed")
	type args struct{ q fakeQuerier }
	type want struct {
		id  int64
		err error // errors.Is で照合。nil は成功
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
		err error // errors.Is で照合。nil は成功
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
