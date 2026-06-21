package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type MemberStub struct {
	Members []db.MemberMember
	Member  db.MemberMember
	Err     error
}

func (s MemberStub) ListMembers(context.Context) ([]db.MemberMember, error) {
	return s.Members, s.Err
}

func (s MemberStub) GetMember(context.Context, int64) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s MemberStub) CreateMember(context.Context, db.CreateMemberParams) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s MemberStub) UpdateMember(context.Context, db.UpdateMemberParams) (db.MemberMember, error) {
	return s.Member, s.Err
}
