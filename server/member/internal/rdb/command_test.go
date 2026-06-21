package rdb

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

func expiresIn(d time.Duration) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now().Add(d), Valid: true}
}

func TestCreateMember(t *testing.T) {
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

func TestUpdateMember(t *testing.T) {
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

func TestCreateSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberCommand(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "stored-hash")

	t.Run("正常系 作成した行を返す", func(t *testing.T) {
		got, err := r.CreateSession(t.Context(), db.CreateSessionParams{
			ID: "hash-new", MemberID: memberID, ExpiresAt: expiresIn(time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		assert.DeepEqual(t, db.MemberSession{ID: "hash-new", MemberID: memberID}, got, "ExpiresAt", "CreatedAt")
	})
	t.Run("準正常系 同一 id は ErrConflict", func(t *testing.T) {
		p := db.CreateSessionParams{ID: "hash-dup", MemberID: memberID, ExpiresAt: expiresIn(time.Hour)}
		if _, err := r.CreateSession(t.Context(), p); err != nil {
			t.Fatalf("setup CreateSession: %v", err)
		}
		if _, err := r.CreateSession(t.Context(), p); !errors.Is(err, dberr.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})
}

func TestDeleteSession(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberCommand(pool)
	q := NewMemberQuery(pool)
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
		if _, err := q.GetSession(t.Context(), "hash-live"); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("削除後 GetSession err = %v, want ErrNotFound", err)
		}
	})
	t.Run("準正常系 未存在の削除は冪等 (error なし)", func(t *testing.T) {
		if err := r.DeleteSession(t.Context(), "hash-missing"); err != nil {
			t.Fatalf("DeleteSession (未存在): %v", err)
		}
	})
}
