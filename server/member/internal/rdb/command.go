package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type MemberCommand struct {
	q db.Querier
}

func NewMemberCommand(pool *pgxpool.Pool) *MemberCommand {
	return &MemberCommand{q: db.New(pool)}
}

func (r *MemberCommand) CreateMember(ctx context.Context, arg db.CreateMemberParams) (db.MemberMember, error) {
	row, err := r.q.CreateMember(ctx, arg)
	if err != nil {
		return db.MemberMember{}, dberr.FromWrite(err)
	}
	return row, nil
}

func (r *MemberCommand) UpdateMember(ctx context.Context, arg db.UpdateMemberParams) (db.MemberMember, error) {
	row, err := r.q.UpdateMember(ctx, arg)
	if err != nil {
		return db.MemberMember{}, dberr.FromUpdate(err)
	}
	return row, nil
}
