package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
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
		out = append(out, toAPIPayment(r))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) GetPayment(c *gin.Context, id api.IdPath) {
	row, err := h.repo.GetPayment(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("payment not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIPayment(row))
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req api.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.AmountCents < 0 {
		_ = c.Error(middleware.Unprocessable("amountCents must not be negative"))
		return
	}
	row, err := h.repo.CreatePayment(c.Request.Context(), db.CreatePaymentParams{
		OrderID:     req.OrderId,
		AmountCents: req.AmountCents,
		Method:      req.Method,
		Status:      req.Status,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIPayment(row))
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
