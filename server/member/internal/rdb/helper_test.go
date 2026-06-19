package rdb

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

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
