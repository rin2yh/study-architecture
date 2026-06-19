package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

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

func (h *Handler) UpdatePayment(c *gin.Context, id api.IdPath) {
	var req api.UpdatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.repo.UpdatePayment(c.Request.Context(), db.UpdatePaymentParams{
		ID:     id,
		Status: req.Status,
	})
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
