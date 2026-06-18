package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても実 SQL が
// schema と噛み合うかは検証できない。DATABASE_URL_CUSTOMER が指す実 DB (compose の
// db-customer / CI の service) へ接続して結合テストする。skip 条件は testdb 参照。
const dbEnv = "DATABASE_URL_CUSTOMER"

func seedPayments(t *testing.T, pool *pgxpool.Pool, rows ...db.PaymentPayment) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE payment.payments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO payment.payments (order_id, amount_cents, method, status) VALUES ($1, $2, $3, $4)`,
			r.OrderID, r.AmountCents, r.Method, r.Status); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func TestRepositoryListPayments(t *testing.T) {
	type args struct {
		seed []db.PaymentPayment
	}
	type want struct {
		methods []string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			args: args{seed: []db.PaymentPayment{
				{OrderID: 1, AmountCents: 1980, Method: "card", Status: "paid"},
				{OrderID: 2, AmountCents: 2980, Method: "bank", Status: "pending"},
			}},
			want: want{methods: []string{"card", "bank"}},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			args: args{seed: nil},
			want: want{methods: []string{}},
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedPayments(t, pool, tt.args.seed...)

			got, err := r.ListPayments(context.Background())
			if err != nil {
				t.Fatalf("ListPayments: %v", err)
			}
			if got == nil {
				t.Fatal("ListPayments: want non-nil slice (emit_empty_slices)")
			}
			if len(got) != len(tt.want.methods) {
				t.Fatalf("len = %d, want %d (%+v)", len(got), len(tt.want.methods), got)
			}
			for i, method := range tt.want.methods {
				if got[i].Method != method {
					t.Fatalf("rows[%d].Method = %q, want %q", i, got[i].Method, method)
				}
			}
		})
	}
}

func TestRepositoryListPaymentsError(t *testing.T) {
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ListPayments(ctx); err == nil {
		t.Fatal("ListPayments: want error from canceled context")
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
