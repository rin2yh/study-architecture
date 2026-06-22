package handler

import (
	"errors"
	"log/slog"
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
	row, err := h.command.UpdatePayment(c.Request.Context(), db.UpdatePaymentParams{
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

	if event.IsSettled(row.Status) {
		if err := h.publisher.PublishPaymentSettled(c.Request.Context(), paymentevent.Settled{
			PaymentID:   row.ID,
			OrderID:     row.OrderID,
			AmountCents: row.AmountCents,
		}); err != nil {
			// 決済確定自体は確定済み。outbox の無い Step 0 ではイベントを再送できないため
			// 可視化のみ行い 200 を返す (ADR-[[202606211200]] の結果整合の宿題)。
			slog.Error("publish payment.settled failed", "paymentId", row.ID, "orderId", row.OrderID, "error", err)
		}
	}

	c.JSON(http.StatusOK, toAPIPayment(row))
}
