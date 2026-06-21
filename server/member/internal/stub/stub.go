package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type RDB struct {
	Members []db.MemberMember
	Member  db.MemberMember
	Err     error
}

func (s RDB) ListMembers(context.Context) ([]db.MemberMember, error) {
	return s.Members, s.Err
}

func (s RDB) GetMember(context.Context, int64) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s RDB) CreateMember(context.Context, db.CreateMemberParams) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s RDB) UpdateMember(context.Context, db.UpdateMemberParams) (db.MemberMember, error) {
	return s.Member, s.Err
}
