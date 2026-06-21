package rdb

import (
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

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
