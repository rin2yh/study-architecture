package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても実 SQL が
// schema と噛み合うかは検証できない。DATABASE_URL_CUSTOMER が指す実 DB (compose の
// db-customer / CI の service) へ接続して結合テストする。skip 条件は testdb 参照。
const dbEnv = "DATABASE_URL_CUSTOMER"

func seedMembers(t *testing.T, pool *pgxpool.Pool, rows ...db.MemberMember) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO member.members (email, display_name) VALUES ($1, $2)`,
			r.Email, r.DisplayName); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func TestRepositoryListMembers(t *testing.T) {
	type args struct {
		seed []db.MemberMember
	}
	type want struct {
		emails []string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			args: args{seed: []db.MemberMember{
				{Email: "a@example.com", DisplayName: "会員A"},
				{Email: "b@example.com", DisplayName: "会員B"},
			}},
			want: want{emails: []string{"a@example.com", "b@example.com"}},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			args: args{seed: nil},
			want: want{emails: []string{}},
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedMembers(t, pool, tt.args.seed...)

			got, err := r.ListMembers(context.Background())
			if err != nil {
				t.Fatalf("ListMembers: %v", err)
			}
			if got == nil {
				t.Fatal("ListMembers: want non-nil slice (emit_empty_slices)")
			}
			if len(got) != len(tt.want.emails) {
				t.Fatalf("len = %d, want %d (%+v)", len(got), len(tt.want.emails), got)
			}
			for i, email := range tt.want.emails {
				if got[i].Email != email {
					t.Fatalf("rows[%d].Email = %q, want %q", i, got[i].Email, email)
				}
			}
		})
	}
}

func TestRepositoryListMembersError(t *testing.T) {
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ListMembers(ctx); err == nil {
		t.Fatal("ListMembers: want error from canceled context")
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
