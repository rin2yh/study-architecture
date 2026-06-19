package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

// seedOneMember は session の FK 先となる会員を 1 件作り、その id を返す。
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

func TestRepositoryGetMemberByEmail(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	seedOneMember(t, pool, "user@example.com", "stored-hash")

	t.Run("正常系 email で会員を引け password_hash も載る", func(t *testing.T) {
		got, err := r.GetMemberByEmail(t.Context(), "user@example.com")
		if err != nil {
			t.Fatalf("GetMemberByEmail: %v", err)
		}
		if got.Email != "user@example.com" || got.PasswordHash != "stored-hash" {
			t.Fatalf("unexpected member: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetMemberByEmail(t.Context(), "none@example.com"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestRepositorySessionLifecycle(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "stored-hash")

	ts := func(d time.Duration) pgtype.Timestamptz {
		return pgtype.Timestamptz{Time: time.Now().Add(d), Valid: true}
	}

	t.Run("正常系 作成→取得→破棄", func(t *testing.T) {
		_, err := r.CreateSession(t.Context(), db.CreateSessionParams{
			ID: "hash-live", MemberID: memberID, ExpiresAt: ts(time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		got, err := r.GetSession(t.Context(), "hash-live")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got.MemberID != memberID {
			t.Fatalf("memberID = %d, want %d", got.MemberID, memberID)
		}
		if err := r.DeleteSession(t.Context(), "hash-live"); err != nil {
			t.Fatalf("DeleteSession: %v", err)
		}
		if _, err := r.GetSession(t.Context(), "hash-live"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("破棄後 GetSession err = %v, want ErrNotFound", err)
		}
	})

	t.Run("異常系 期限切れは取得できず ErrNotFound", func(t *testing.T) {
		if _, err := r.CreateSession(t.Context(), db.CreateSessionParams{
			ID: "hash-expired", MemberID: memberID, ExpiresAt: ts(-time.Hour),
		}); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if _, err := r.GetSession(t.Context(), "hash-expired"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("準正常系 未存在の破棄は冪等 (error なし)", func(t *testing.T) {
		if err := r.DeleteSession(t.Context(), "hash-missing"); err != nil {
			t.Fatalf("DeleteSession (未存在): %v", err)
		}
	})
}
