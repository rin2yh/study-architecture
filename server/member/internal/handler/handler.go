package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
)

type Query interface {
	ListMembers(ctx context.Context) ([]db.MemberMember, error)
	GetMember(ctx context.Context, id int64) (db.MemberMember, error)
}

type Command interface {
	CreateMember(ctx context.Context, arg db.CreateMemberParams) (db.MemberMember, error)
	UpdateMember(ctx context.Context, arg db.UpdateMemberParams) (db.MemberMember, error)
}

type readHandler struct {
	query Query
}

type writeHandler struct {
	command Command
}

// Handler は readHandler / writeHandler を束ねて単一の api.ServerInterface を満たす。
// 各サブハンドラは Query / Command の片側にしか依存しないため、読み書きの依存が型で分離される。
type Handler struct {
	*readHandler
	*writeHandler
}

var _ api.ServerInterface = (*Handler)(nil)

func New(query Query, command Command) *Handler {
	return &Handler{
		readHandler:  &readHandler{query: query},
		writeHandler: &writeHandler{command: command},
	}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func toAPIMember(r db.MemberMember) api.Member {
	return api.Member{
		Id:          r.ID,
		Email:       r.Email,
		DisplayName: r.DisplayName,
		CreatedAt:   r.CreatedAt.Time,
	}
}
