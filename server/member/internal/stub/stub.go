package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type Repo struct {
	Members []db.MemberMember
	Member  db.MemberMember
	Err     error
}

func (s Repo) ListMembers(context.Context) ([]db.MemberMember, error) {
	return s.Members, s.Err
}

func (s Repo) GetMember(context.Context, int64) (db.MemberMember, error) {
	return s.Member, s.Err
}

func (s Repo) CreateMember(context.Context, db.CreateMemberParams) (db.MemberMember, error) {
	return s.Member, s.Err
}
