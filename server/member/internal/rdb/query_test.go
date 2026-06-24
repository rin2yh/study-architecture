package rdb

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

const dbEnv = "DATABASE_URL_MEMBER"

func seedMembers(t *testing.T, pool *pgxpool.Pool, rows ...db.MemberMember) {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY CASCADE`); err != nil {
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

func seedOneMember(t *testing.T, pool *pgxpool.Pool, email, hash string) int64 {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE member.members RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	var id int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO member.members (email, display_name, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		email, "会員", hash).Scan(&id); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	return id
}

func TestListMembers(t *testing.T) {
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
			assert.DeepEqualSlice(t, tt.seed, got, "ID", "CreatedAt")
		})
	}
}

func TestListMembersError(t *testing.T) {
	skip.Short(t)
	r := NewMemberQuery(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListMembers(ctx); err == nil {
		t.Fatal("ListMembers: want error from canceled context")
	}
}

func TestGetMember(t *testing.T) {
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
	t.Run("準正常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetMember(t.Context(), 9999); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestGetMemberByEmail(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberQuery(pool)
	seedOneMember(t, pool, "user@example.com", "stored-hash")

	t.Run("正常系 email で会員を引け password_hash も載る", func(t *testing.T) {
		got, err := r.GetMemberByEmail(t.Context(), "user@example.com")
		if err != nil {
			t.Fatalf("GetMemberByEmail: %v", err)
		}
		assert.DeepEqual(t, db.MemberMember{Email: "user@example.com", DisplayName: "会員", PasswordHash: "stored-hash"}, got, "ID", "CreatedAt")
	})
	t.Run("準正常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetMemberByEmail(t.Context(), "none@example.com"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestGetSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberQuery(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "stored-hash")
	ctx := t.Context()
	if _, err := pool.Exec(ctx,
		`INSERT INTO member.sessions (id, member_id, expires_at) VALUES
		 ('live', $1, now() + interval '1 hour'),
		 ('expired', $1, now() - interval '1 hour')`, memberID); err != nil {
		t.Fatalf("insert sessions: %v", err)
	}

	t.Run("正常系 有効なセッションを返す", func(t *testing.T) {
		got, err := r.GetSession(t.Context(), "live")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		assert.DeepEqual(t, db.MemberSession{ID: "live", MemberID: memberID}, got, "ExpiresAt", "CreatedAt")
	})
	t.Run("準正常系 期限切れは ErrNotFound", func(t *testing.T) {
		if _, err := r.GetSession(t.Context(), "expired"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
	t.Run("準正常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetSession(t.Context(), "missing"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
