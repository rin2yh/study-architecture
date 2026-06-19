package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/db"
	"github.com/rin2yh/study-architecture/server/member/internal/repository"
)

type Handler struct {
	repo repository.MemberRepository
}

var _ api.ServerInterface = (*Handler)(nil)

func New(repo repository.MemberRepository) *Handler {
	return &Handler{repo: repo}
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

func toAPISession(token string, r db.MemberSession) api.Session {
	return api.Session{
		Id:        token,
		MemberId:  r.MemberID,
		ExpiresAt: r.ExpiresAt.Time,
	}
}
