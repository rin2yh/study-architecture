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

func (r *MemberCommand) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.MemberSession, error) {
	row, err := r.q.CreateSession(ctx, arg)
	if err != nil {
		return db.MemberSession{}, dberr.FromWrite(err)
	}
	return row, nil
}

func (r *MemberCommand) DeleteSession(ctx context.Context, id string) error {
	return r.q.DeleteSession(ctx, id)
}

func (r *MemberCommand) CreateAddress(ctx context.Context, arg db.CreateAddressParams) (db.MemberAddress, error) {
	row, err := r.q.CreateAddress(ctx, arg)
	if err != nil {
		return db.MemberAddress{}, dberr.FromWrite(err)
	}
	return row, nil
}

func (r *MemberCommand) UpdateAddress(ctx context.Context, arg db.UpdateAddressParams) (db.MemberAddress, error) {
	row, err := r.q.UpdateAddress(ctx, arg)
	if err != nil {
		return db.MemberAddress{}, dberr.FromUpdate(err)
	}
	return row, nil
}

func (r *MemberCommand) DeleteAddress(ctx context.Context, arg db.DeleteAddressParams) error {
	return r.q.DeleteAddress(ctx, arg)
}
