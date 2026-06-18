package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/testdb"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても実 SQL が
// schema と噛み合うかは検証できない。DATABASE_URL_CUSTOMER が指す実 DB (compose の
// db-customer / CI の service) へ接続して結合テストする。skip 条件は testdb 参照。
const dbEnv = "DATABASE_URL_CUSTOMER"

// seedOrders は table を空にしてから rows を id 昇順 (= 挿入順) で入れ直す。
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
	type args struct {
		seed []db.OrderOrder
	}
	type want struct {
		statuses []string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			args: args{seed: []db.OrderOrder{
				{MemberID: 1, Status: "pending", TotalCents: 1980},
				{MemberID: 2, Status: "paid", TotalCents: 2980},
			}},
			want: want{statuses: []string{"pending", "paid"}},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			args: args{seed: nil},
			want: want{statuses: []string{}},
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedOrders(t, pool, tt.args.seed...)

			got, err := r.ListOrders(context.Background())
			if err != nil {
				t.Fatalf("ListOrders: %v", err)
			}
			if got == nil {
				t.Fatal("ListOrders: want non-nil slice (emit_empty_slices)")
			}
			if len(got) != len(tt.want.statuses) {
				t.Fatalf("len = %d, want %d (%+v)", len(got), len(tt.want.statuses), got)
			}
			for i, status := range tt.want.statuses {
				if got[i].Status != status {
					t.Fatalf("rows[%d].Status = %q, want %q", i, got[i].Status, status)
				}
			}
		})
	}
}

// 異常系: 接続不能 (閉じた pool) でクエリがエラーを伝播することを確認する。
func TestRepositoryListOrdersError(t *testing.T) {
	r := NewRepository(testdb.OpenClosed(t, dbEnv))
	if _, err := r.ListOrders(context.Background()); err == nil {
		t.Fatal("ListOrders: want error from closed pool")
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
