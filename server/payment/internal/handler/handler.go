package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/repository"
)

type Handler struct {
	repo repository.PaymentRepository
}

var _ api.ServerInterface = (*Handler)(nil)

func New(repo repository.PaymentRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ListPayments(c *gin.Context) {
	rows, err := h.repo.ListPayments(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]api.Payment, 0, len(rows))
	for _, r := range rows {
		out = append(out, api.Payment{
			Id:          r.ID,
			OrderId:     r.OrderID,
			AmountCents: r.AmountCents,
			Method:      r.Method,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt.Time,
		})
	}
	c.JSON(http.StatusOK, out)
}
