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

func (r *MemberQuery) ListMembers(ctx context.Context) ([]Member, error) {
	rows, err := r.q.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	return toMembers(rows), nil
}

func (r *MemberQuery) GetMember(ctx context.Context, id int64) (Member, error) {
	row, err := r.q.GetMember(ctx, id)
	if err != nil {
		return Member{}, dberr.FromRead(err)
	}
	return toMember(row), nil
}

func (r *MemberQuery) GetMemberByEmail(ctx context.Context, email string) (Member, error) {
	row, err := r.q.GetMemberByEmail(ctx, email)
	if err != nil {
		return Member{}, dberr.FromRead(err)
	}
	return toMember(row), nil
}

func (r *MemberQuery) GetSession(ctx context.Context, id string) (Session, error) {
	row, err := r.q.GetSession(ctx, id)
	if err != nil {
		return Session{}, dberr.FromRead(err)
	}
	return toSession(row), nil
}

func (r *MemberQuery) ListAddresses(ctx context.Context, memberID int64) ([]Address, error) {
	rows, err := r.q.ListAddresses(ctx, memberID)
	if err != nil {
		return nil, err
	}
	return toAddresses(rows), nil
}

func (r *MemberQuery) GetAddress(ctx context.Context, ref AddressRef) (Address, error) {
	row, err := r.q.GetAddress(ctx, db.GetAddressParams{ID: ref.ID, MemberID: ref.MemberID})
	if err != nil {
		return Address{}, dberr.FromRead(err)
	}
	return toAddress(row), nil
}
