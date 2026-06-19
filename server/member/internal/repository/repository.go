package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type MemberRepository interface {
	ListMembers(ctx context.Context) ([]db.MemberMember, error)
	GetMember(ctx context.Context, id int64) (db.MemberMember, error)
	CreateMember(ctx context.Context, arg db.CreateMemberParams) (db.MemberMember, error)
	UpdateMember(ctx context.Context, arg db.UpdateMemberParams) (db.MemberMember, error)
}

type Repository struct {
	q db.Querier
}

var _ MemberRepository = (*Repository)(nil)

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

func (r *Repository) GetMember(ctx context.Context, id int64) (db.MemberMember, error) {
	row, err := r.q.GetMember(ctx, id)
	if err != nil {
		return db.MemberMember{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *Repository) CreateMember(ctx context.Context, arg db.CreateMemberParams) (db.MemberMember, error) {
	row, err := r.q.CreateMember(ctx, arg)
	if err != nil {
		return db.MemberMember{}, dberr.FromWrite(err)
	}
	return row, nil
}

func (r *Repository) UpdateMember(ctx context.Context, arg db.UpdateMemberParams) (db.MemberMember, error) {
	row, err := r.q.UpdateMember(ctx, arg)
	if err != nil {
		return db.MemberMember{}, dberr.FromUpdate(err)
	}
	return row, nil
}
