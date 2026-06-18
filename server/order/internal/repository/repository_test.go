package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても実 SQL が
// schema と噛み合うかは検証できない。List は DATABASE_URL_CUSTOMER が指す実 DB へ接続して
// 結合テストする。skip 条件は testdb 参照。
const dbEnv = "DATABASE_URL_CUSTOMER"

// fakeQuerier は Get/Create のエラー正規化 (dberr) を DB なしで検証するための注入点。
type fakeQuerier struct {
	rows  []db.OrderOrder
	order db.OrderOrder
	err   error
}

func (f fakeQuerier) ListOrders(context.Context) ([]db.OrderOrder, error) {
	return f.rows, f.err
}

func (f fakeQuerier) GetOrder(context.Context, int64) (db.OrderOrder, error) {
	return f.order, f.err
}

func (f fakeQuerier) CreateOrder(context.Context, db.CreateOrderParams) (db.OrderOrder, error) {
	return f.order, f.err
}

func seedOrders(t *testing.T, pool *pgxpool.Pool, rows ...db.OrderOrder) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO "order".orders (member_id, status, total_cents) VALUES ($1, $2, $3)`,
			r.MemberID, r.Status, r.TotalCents); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func TestRepositoryListOrders(t *testing.T) {
	tests := []struct {
		name string
		seed []db.OrderOrder
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			seed: []db.OrderOrder{
				{MemberID: 1, Status: "pending", TotalCents: 1980},
				{MemberID: 2, Status: "paid", TotalCents: 2980},
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
			seedOrders(t, pool, tt.seed...)

			got, err := r.ListOrders(context.Background())
			if err != nil {
				t.Fatalf("ListOrders: %v", err)
			}
			if got == nil {
				t.Fatal("ListOrders: want non-nil slice (emit_empty_slices)")
			}
			if diff := cmp.Diff(tt.seed, got,
				cmpopts.IgnoreFields(db.OrderOrder{}, "ID", "CreatedAt"),
				cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListOrders mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryListOrdersError(t *testing.T) {
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ListOrders(ctx); err == nil {
		t.Fatal("ListOrders: want error from canceled context")
	}
}

func TestRepositoryGetOrder(t *testing.T) {
	order := db.OrderOrder{ID: 1, MemberID: 10, Status: "paid"}
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
		{"正常系 行を返す", args{fakeQuerier{order: order}}, want{1, nil}},
		{"異常系 no rows は ErrNotFound に正規化", args{fakeQuerier{err: pgx.ErrNoRows}}, want{0, dberr.ErrNotFound}},
		{"異常系 その他エラーは透過", args{fakeQuerier{err: other}}, want{0, other}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).GetOrder(context.Background(), 1)
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetOrder: %v", err)
			}
			if got.ID != tt.want.id {
				t.Fatalf("id = %d, want %d", got.ID, tt.want.id)
			}
		})
	}
}

func TestRepositoryCreateOrder(t *testing.T) {
	created := db.OrderOrder{ID: 10, MemberID: 20, Status: "pending"}
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
		{"正常系 作成行を返す", args{fakeQuerier{order: created}}, want{10, nil}},
		{"異常系 その他エラーは透過", args{fakeQuerier{err: other}}, want{0, other}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).CreateOrder(context.Background(), db.CreateOrderParams{})
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateOrder: %v", err)
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
