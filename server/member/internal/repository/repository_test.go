package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

const dbEnv = "DATABASE_URL_CUSTOMER"

type fakeQuerier struct {
	rows   []db.MemberMember
	member db.MemberMember
	err    error
}

func (f fakeQuerier) ListMembers(context.Context) ([]db.MemberMember, error) {
	return f.rows, f.err
}

func (f fakeQuerier) GetMember(context.Context, int64) (db.MemberMember, error) {
	return f.member, f.err
}

func (f fakeQuerier) CreateMember(context.Context, db.CreateMemberParams) (db.MemberMember, error) {
	return f.member, f.err
}

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
	r := NewRepository(pool)
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
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListMembers(ctx); err == nil {
		t.Fatal("ListMembers: want error from canceled context")
	}
}

func TestRepositoryGetMember(t *testing.T) {
	member := db.MemberMember{ID: 1, Email: "user@example.com"}
	type args struct{ q fakeQuerier }
	type want struct {
		id  int64
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 行を返す", args{fakeQuerier{member: member}}, want{1, nil}},
		{"異常系 no rows は ErrNotFound に正規化", args{fakeQuerier{err: pgx.ErrNoRows}}, want{0, dberr.ErrNotFound}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).GetMember(t.Context(), 1)
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetMember: %v", err)
			}
			if got.ID != tt.want.id {
				t.Fatalf("id = %d, want %d", got.ID, tt.want.id)
			}
		})
	}
}

func TestRepositoryCreateMember(t *testing.T) {
	created := db.MemberMember{ID: 10, Email: "new@example.com"}
	type args struct{ q fakeQuerier }
	type want struct {
		id  int64
		err error
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 作成行を返す", args{fakeQuerier{member: created}}, want{10, nil}},
		{"異常系 unique_violation は ErrConflict に正規化", args{fakeQuerier{err: &pgconn.PgError{Code: "23505"}}}, want{0, dberr.ErrConflict}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).CreateMember(t.Context(), db.CreateMemberParams{})
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateMember: %v", err)
			}
			if got.ID != tt.want.id {
				t.Fatalf("id = %d, want %d", got.ID, tt.want.id)
			}
		})
	}
}
