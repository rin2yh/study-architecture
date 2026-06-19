package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

type Query interface {
	ListOrders(ctx context.Context) ([]db.OrderOrder, error)
	GetOrder(ctx context.Context, id int64) (db.OrderOrder, error)
}

type Command interface {
	CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.OrderOrder, error)
	UpdateOrder(ctx context.Context, arg db.UpdateOrderParams) (db.OrderOrder, error)
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

func toAPIOrder(r db.OrderOrder) api.Order {
	return api.Order{
		Id:         r.ID,
		MemberId:   r.MemberID,
		Status:     r.Status,
		TotalCents: r.TotalCents,
		CreatedAt:  r.CreatedAt.Time,
	}
}
