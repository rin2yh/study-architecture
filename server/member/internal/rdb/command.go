package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
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

func (r *MemberCommand) CreateMember(ctx context.Context, arg MemberCreate) (Member, error) {
	row, err := r.q.CreateMember(ctx, db.CreateMemberParams{
		Email:        arg.Email,
		DisplayName:  arg.DisplayName,
		PasswordHash: arg.PasswordHash,
	})
	if err != nil {
		return Member{}, dberr.FromWrite(err)
	}
	return toMember(row), nil
}

func (r *MemberCommand) UpdateMember(ctx context.Context, arg MemberUpdate) (Member, error) {
	row, err := r.q.UpdateMember(ctx, db.UpdateMemberParams{
		ID:          arg.ID,
		Email:       arg.Email,
		DisplayName: arg.DisplayName,
	})
	if err != nil {
		return Member{}, dberr.FromUpdate(err)
	}
	return toMember(row), nil
}

func (r *MemberCommand) CreateSession(ctx context.Context, arg SessionCreate) (Session, error) {
	row, err := r.q.CreateSession(ctx, db.CreateSessionParams{
		ID:        arg.ID,
		MemberID:  arg.MemberID,
		ExpiresAt: pgtype.Timestamptz{Time: arg.ExpiresAt, Valid: true},
	})
	if err != nil {
		return Session{}, dberr.FromWrite(err)
	}
	return toSession(row), nil
}

func (r *MemberCommand) DeleteSession(ctx context.Context, id string) error {
	return r.q.DeleteSession(ctx, id)
}

func (r *MemberCommand) CreateAddress(ctx context.Context, arg AddressCreate) (Address, error) {
	row, err := r.q.CreateAddress(ctx, db.CreateAddressParams{
		MemberID:   arg.MemberID,
		Recipient:  arg.Recipient,
		PostalCode: arg.PostalCode,
		Prefecture: arg.Prefecture,
		City:       arg.City,
		Line1:      arg.Line1,
	})
	if err != nil {
		return Address{}, dberr.FromWriteFK(err)
	}
	return toAddress(row), nil
}

func (r *MemberCommand) UpdateAddress(ctx context.Context, arg AddressUpdate) (Address, error) {
	row, err := r.q.UpdateAddress(ctx, db.UpdateAddressParams{
		ID:         arg.ID,
		MemberID:   arg.MemberID,
		Recipient:  arg.Recipient,
		PostalCode: arg.PostalCode,
		Prefecture: arg.Prefecture,
		City:       arg.City,
		Line1:      arg.Line1,
	})
	if err != nil {
		return Address{}, dberr.FromUpdate(err)
	}
	return toAddress(row), nil
}

func (r *MemberCommand) DeleteAddress(ctx context.Context, ref AddressRef) error {
	return r.q.DeleteAddress(ctx, db.DeleteAddressParams{ID: ref.ID, MemberID: ref.MemberID})
}
