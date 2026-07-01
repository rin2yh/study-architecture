package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/member/internal/rdb"
)

type MemberStub struct {
	Members []rdb.Member
	Member  rdb.Member
	Err     error

	Session    rdb.Session
	SessionErr error

	Addresses  []rdb.Address
	Address    rdb.Address
	AddressErr error
}

func (s MemberStub) ListMembers(context.Context) ([]rdb.Member, error) {
	return s.Members, s.Err
}

func (s MemberStub) GetMember(context.Context, int64) (rdb.Member, error) {
	return s.Member, s.Err
}

func (s MemberStub) GetMemberByEmail(context.Context, string) (rdb.Member, error) {
	return s.Member, s.Err
}

func (s MemberStub) CreateMember(context.Context, rdb.MemberCreate) (rdb.Member, error) {
	return s.Member, s.Err
}

func (s MemberStub) UpdateMember(context.Context, rdb.MemberUpdate) (rdb.Member, error) {
	return s.Member, s.Err
}

func (s MemberStub) CreateSession(context.Context, rdb.SessionCreate) (rdb.Session, error) {
	return s.Session, s.SessionErr
}

func (s MemberStub) GetSession(context.Context, string) (rdb.Session, error) {
	return s.Session, s.SessionErr
}

func (s MemberStub) DeleteSession(context.Context, string) error {
	return s.SessionErr
}

func (s MemberStub) ListAddresses(context.Context, int64) ([]rdb.Address, error) {
	return s.Addresses, s.AddressErr
}

func (s MemberStub) GetAddress(context.Context, rdb.AddressRef) (rdb.Address, error) {
	return s.Address, s.AddressErr
}

func (s MemberStub) CreateAddress(context.Context, rdb.AddressCreate) (rdb.Address, error) {
	return s.Address, s.AddressErr
}

func (s MemberStub) UpdateAddress(context.Context, rdb.AddressUpdate) (rdb.Address, error) {
	return s.Address, s.AddressErr
}

func (s MemberStub) DeleteAddress(context.Context, rdb.AddressRef) error {
	return s.AddressErr
}
