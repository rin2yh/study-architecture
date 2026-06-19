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

type Handler struct {
	query   Query
	command Command
}

var _ api.ServerInterface = (*Handler)(nil)

func New(query Query, command Command) *Handler {
	return &Handler{query: query, command: command}
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
