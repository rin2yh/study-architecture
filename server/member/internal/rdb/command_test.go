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
	t.Run("準正常系 email 重複は ErrConflict", func(t *testing.T) {
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
	t.Run("準正常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateMember(t.Context(), db.UpdateMemberParams{ID: 9999, Email: "x@example.com", DisplayName: "x"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
	t.Run("準正常系 email 重複は ErrConflict", func(t *testing.T) {
		if _, err := r.UpdateMember(t.Context(), db.UpdateMemberParams{ID: 1, Email: "b@example.com", DisplayName: "衝突"}); !errors.Is(err, dberr.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})
}

func TestCreateAddress(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberCommand(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "h")
	seedAddresses(t, pool, memberID)

	t.Run("正常系 作成行を返す", func(t *testing.T) {
		got, err := r.CreateAddress(t.Context(), db.CreateAddressParams{
			MemberID: memberID, Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3",
		})
		if err != nil {
			t.Fatalf("CreateAddress: %v", err)
		}
		if got.ID == 0 || got.MemberID != memberID || got.Recipient != "山田太郎" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
}

func TestUpdateAddress(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberCommand(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "h")
	seedAddresses(t, pool, memberID, db.MemberAddress{Recipient: "旧名", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "旧1-2-3"})

	t.Run("正常系 member 所有の住所を更新して返す", func(t *testing.T) {
		got, err := r.UpdateAddress(t.Context(), db.UpdateAddressParams{
			ID: 1, MemberID: memberID, Recipient: "新名", PostalCode: "1000001", Prefecture: "東京都", City: "千代田区", Line1: "丸の内1-1-1",
		})
		if err != nil {
			t.Fatalf("UpdateAddress: %v", err)
		}
		if got.Recipient != "新名" || got.City != "千代田区" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("準正常系 他 member の住所は更新できず ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateAddress(t.Context(), db.UpdateAddressParams{ID: 1, MemberID: memberID + 1, Recipient: "x", PostalCode: "x", Prefecture: "x", City: "x", Line1: "x"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestDeleteAddress(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewMemberCommand(pool)
	q := NewMemberQuery(pool)
	memberID := seedOneMember(t, pool, "user@example.com", "h")
	seedAddresses(t, pool, memberID, db.MemberAddress{Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"})

	t.Run("正常系 削除すると取得できなくなる", func(t *testing.T) {
		if err := r.DeleteAddress(t.Context(), db.DeleteAddressParams{ID: 1, MemberID: memberID}); err != nil {
			t.Fatalf("DeleteAddress: %v", err)
		}
		if _, err := q.GetAddress(t.Context(), db.GetAddressParams{ID: 1, MemberID: memberID}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("削除後 GetAddress err = %v, want ErrNotFound", err)
		}
	})
	t.Run("準正常系 未存在の削除は冪等 (error なし)", func(t *testing.T) {
		if err := r.DeleteAddress(t.Context(), db.DeleteAddressParams{ID: 9999, MemberID: memberID}); err != nil {
			t.Fatalf("DeleteAddress (未存在): %v", err)
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
