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

func expiresIn(d time.Duration) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now().Add(d), Valid: true}
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

func TestRepositoryCreateSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "stored-hash")

	t.Run("正常系 作成した行を返す", func(t *testing.T) {
		got, err := r.CreateSession(t.Context(), db.CreateSessionParams{
			ID: "hash-new", MemberID: memberID, ExpiresAt: expiresIn(time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if got.ID != "hash-new" || got.MemberID != memberID {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 同一 id は ErrConflict", func(t *testing.T) {
		p := db.CreateSessionParams{ID: "hash-dup", MemberID: memberID, ExpiresAt: expiresIn(time.Hour)}
		if _, err := r.CreateSession(t.Context(), p); err != nil {
			t.Fatalf("setup CreateSession: %v", err)
		}
		if _, err := r.CreateSession(t.Context(), p); !errors.Is(err, dberr.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})
}

func TestRepositoryGetSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "stored-hash")
	mustCreate := func(id string, d time.Duration) {
		t.Helper()
		if _, err := r.CreateSession(t.Context(), db.CreateSessionParams{ID: id, MemberID: memberID, ExpiresAt: expiresIn(d)}); err != nil {
			t.Fatalf("seed session %s: %v", id, err)
		}
	}
	mustCreate("hash-live", time.Hour)
	mustCreate("hash-expired", -time.Hour)

	t.Run("正常系 有効なセッションを返す", func(t *testing.T) {
		got, err := r.GetSession(t.Context(), "hash-live")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got.MemberID != memberID {
			t.Fatalf("memberID = %d, want %d", got.MemberID, memberID)
		}
	})
	t.Run("準正常系 期限切れは ErrNotFound", func(t *testing.T) {
		if _, err := r.GetSession(t.Context(), "hash-expired"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetSession(t.Context(), "hash-missing"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestRepositoryDeleteSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "stored-hash")
	if _, err := r.CreateSession(t.Context(), db.CreateSessionParams{
		ID: "hash-live", MemberID: memberID, ExpiresAt: expiresIn(time.Hour),
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	t.Run("正常系 削除すると取得できなくなる", func(t *testing.T) {
		if err := r.DeleteSession(t.Context(), "hash-live"); err != nil {
			t.Fatalf("DeleteSession: %v", err)
		}
		if _, err := r.GetSession(t.Context(), "hash-live"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("削除後 GetSession err = %v, want ErrNotFound", err)
		}
	})
	t.Run("準正常系 未存在の削除は冪等 (error なし)", func(t *testing.T) {
		if err := r.DeleteSession(t.Context(), "hash-missing"); err != nil {
			t.Fatalf("DeleteSession (未存在): %v", err)
		}
	})
}
