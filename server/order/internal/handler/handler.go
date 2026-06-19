package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
)

type Handler struct {
	repo repository.OrderRepository
}

var _ api.ServerInterface = (*Handler)(nil)

func New(repo repository.OrderRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func toAPIOrder(r db.OrderOrder) api.Order {
	return api.Order{
		Id:         r.ID,
		MemberId:   r.MemberID,
		Status:     r.Status,
		TotalCents: r.TotalCents,
		CreatedAt:  r.CreatedAt.Time,
	}
}
