package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type MemberQuery struct {
	q db.Querier
}

func NewMemberQuery(pool *pgxpool.Pool) *MemberQuery {
	return &MemberQuery{q: db.New(pool)}
}

func (r *MemberQuery) ListMembers(ctx context.Context) ([]db.MemberMember, error) {
	return r.q.ListMembers(ctx)
}

func (r *MemberQuery) GetMember(ctx context.Context, id int64) (db.MemberMember, error) {
	row, err := r.q.GetMember(ctx, id)
	if err != nil {
		return db.MemberMember{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *MemberQuery) GetMemberByEmail(ctx context.Context, email string) (db.MemberMember, error) {
	row, err := r.q.GetMemberByEmail(ctx, email)
	if err != nil {
		return db.MemberMember{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *MemberQuery) GetSession(ctx context.Context, id string) (db.MemberSession, error) {
	row, err := r.q.GetSession(ctx, id)
	if err != nil {
		return db.MemberSession{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *MemberQuery) ListAddresses(ctx context.Context, memberID int64) ([]db.MemberAddress, error) {
	return r.q.ListAddresses(ctx, memberID)
}

func (r *MemberQuery) GetAddress(ctx context.Context, arg db.GetAddressParams) (db.MemberAddress, error) {
	row, err := r.q.GetAddress(ctx, arg)
	if err != nil {
		return db.MemberAddress{}, dberr.FromRead(err)
	}
	return row, nil
}
