package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type MemberStub struct {
	Members []db.MemberMember
	Member  db.MemberMember
	Err     error

	Session    db.MemberSession
	SessionErr error
}

func (s MemberStub) ListMembers(context.Context) ([]db.MemberMember, error) {
	return s.Members, s.Err
}

func (s MemberStub) GetMember(context.Context, int64) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s MemberStub) GetMemberByEmail(context.Context, string) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s MemberStub) CreateMember(context.Context, db.CreateMemberParams) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s MemberStub) UpdateMember(context.Context, db.UpdateMemberParams) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s MemberStub) CreateSession(context.Context, db.CreateSessionParams) (db.MemberSession, error) {
	return s.Session, s.SessionErr
}

func (s MemberStub) GetSession(context.Context, string) (db.MemberSession, error) {
	return s.Session, s.SessionErr
}

func (s MemberStub) DeleteSession(context.Context, string) error {
	return s.SessionErr
}
