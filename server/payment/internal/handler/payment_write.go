package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
	"github.com/rin2yh/study-architecture/server/payment/internal/event"
)

func (h *writeHandler) CreatePayment(c *gin.Context) {
	var req api.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.AmountCents < 0 {
		_ = c.Error(middleware.Unprocessable("amountCents must not be negative"))
		return
	}
	row, err := h.command.CreatePayment(c.Request.Context(), db.CreatePaymentParams{
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

func (h *writeHandler) UpdatePayment(c *gin.Context, id api.IdPath) {
	var req api.UpdatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	// 確定イベントの送出は後段のリレーに分離する (ADR-[[202606261212]])。
	settled := event.IsSettled(req.Status)
	var traceparent string
	if settled {
		traceparent = paymentevent.Traceparent(c.Request.Context())
	}
	row, err := h.command.UpdatePayment(c.Request.Context(), db.UpdatePaymentParams{
		ID:          id,
		Status:      req.Status,
		MarkSettled: settled,
		Traceparent: traceparent,
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
