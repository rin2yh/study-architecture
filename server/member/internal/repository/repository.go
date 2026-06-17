package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

// interface にすることで handler のテストでスタブへ差し替えられる。
type MemberRepository interface {
	ListMembers(ctx context.Context) ([]db.MemberMember, error)
}

type Repository struct {
	q db.Querier
}

var _ MemberRepository = (*Repository)(nil)

// DI では kessoku.Async でラップし、I/O を伴うこの初期化を並列化対象にしている。
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: db.New(pool)}
}

func (r *Repository) ListMembers(ctx context.Context) ([]db.MemberMember, error) {
	return r.q.ListMembers(ctx)
}
