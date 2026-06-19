package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

type Query interface {
	ListPayments(ctx context.Context) ([]db.PaymentPayment, error)
	GetPayment(ctx context.Context, id int64) (db.PaymentPayment, error)
}

type Command interface {
	CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.PaymentPayment, error)
	UpdatePayment(ctx context.Context, arg db.UpdatePaymentParams) (db.PaymentPayment, error)
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

func toAPIPayment(r db.PaymentPayment) api.Payment {
	return api.Payment{
		Id:          r.ID,
		OrderId:     r.OrderID,
		AmountCents: r.AmountCents,
		Method:      r.Method,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt.Time,
	}
}
