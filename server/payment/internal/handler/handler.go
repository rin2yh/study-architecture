package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/rdb"
)

type Query interface {
	ListPayments(ctx context.Context) ([]rdb.Payment, error)
	GetPayment(ctx context.Context, id int64) (rdb.Payment, error)
}

type Command interface {
	CreatePayment(ctx context.Context, arg rdb.PaymentCreate) (rdb.Payment, error)
	UpdatePayment(ctx context.Context, u rdb.PaymentUpdate) (rdb.Payment, error)
}

type readHandler struct {
	query Query
}

type writeHandler struct {
	command Command
}

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

func toAPIPayment(r rdb.Payment) api.Payment {
	return api.Payment{
		Id:          r.ID,
		OrderId:     r.OrderID,
		AmountCents: r.AmountCents,
		Method:      r.Method,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
	}
}
