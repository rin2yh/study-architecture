package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

const dbEnv = "DATABASE_URL_CUSTOMER"

func seedMembers(t *testing.T, pool *pgxpool.Pool, rows ...db.MemberMember) {
	t.Helper()
	ctx := t.Context()
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
	skip.Short(t)
	tests := []struct {
		name string
		seed []db.MemberMember
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			seed: []db.MemberMember{
				{Email: "a@example.com", DisplayName: "会員A"},
				{Email: "b@example.com", DisplayName: "会員B"},
			},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			seed: nil,
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewMemberQuery(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedMembers(t, pool, tt.seed...)

			got, err := r.ListMembers(t.Context())
			if err != nil {
				t.Fatalf("ListMembers: %v", err)
			}
			if got == nil {
				t.Fatal("ListMembers: want non-nil slice (emit_empty_slices)")
			}
			if diff := cmp.Diff(tt.seed, got,
				cmpopts.IgnoreFields(db.MemberMember{}, "ID", "CreatedAt"),
				cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListMembers mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryListMembersError(t *testing.T) {
	skip.Short(t)
	r := NewMemberQuery(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListMembers(ctx); err == nil {
		t.Fatal("ListMembers: want error from canceled context")
	}
}

func TestRepositoryGetMember(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberQuery(pool)
	seedMembers(t, pool, db.MemberMember{Email: "user@example.com", DisplayName: "会員A"})

	t.Run("正常系 既存 id の行を返す", func(t *testing.T) {
		got, err := r.GetMember(t.Context(), 1)
		if err != nil {
			t.Fatalf("GetMember: %v", err)
		}
		if got.Email != "user@example.com" {
			t.Fatalf("email = %q, want user@example.com", got.Email)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetMember(t.Context(), 9999); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestRepositoryCreateMember(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberCommand(pool)
	seedMembers(t, pool, db.MemberMember{Email: "exist@example.com", DisplayName: "既存"})

	t.Run("正常系 作成行を返す", func(t *testing.T) {
		got, err := r.CreateMember(t.Context(), db.CreateMemberParams{Email: "new@example.com", DisplayName: "新規会員"})
		if err != nil {
			t.Fatalf("CreateMember: %v", err)
		}
		if got.ID == 0 || got.Email != "new@example.com" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 email 重複は ErrConflict", func(t *testing.T) {
		if _, err := r.CreateMember(t.Context(), db.CreateMemberParams{Email: "exist@example.com", DisplayName: "重複"}); !errors.Is(err, dberr.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})
}

func TestRepositoryUpdateMember(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberCommand(pool)
	seedMembers(t, pool,
		db.MemberMember{Email: "a@example.com", DisplayName: "会員A"},
		db.MemberMember{Email: "b@example.com", DisplayName: "会員B"})

	t.Run("正常系 既存行を更新して返す", func(t *testing.T) {
		got, err := r.UpdateMember(t.Context(), db.UpdateMemberParams{ID: 1, Email: "a2@example.com", DisplayName: "会員A更新"})
		if err != nil {
			t.Fatalf("UpdateMember: %v", err)
		}
		if got.ID != 1 || got.Email != "a2@example.com" || got.DisplayName != "会員A更新" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateMember(t.Context(), db.UpdateMemberParams{ID: 9999, Email: "x@example.com", DisplayName: "x"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
	t.Run("異常系 email 重複は ErrConflict", func(t *testing.T) {
		if _, err := r.UpdateMember(t.Context(), db.UpdateMemberParams{ID: 1, Email: "b@example.com", DisplayName: "衝突"}); !errors.Is(err, dberr.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})
}
