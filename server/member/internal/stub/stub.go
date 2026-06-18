package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type Repo struct {
	Members []db.MemberMember
	Err     error
}

func (s Repo) ListMembers(context.Context) ([]db.MemberMember, error) {
	return s.Members, s.Err
}
