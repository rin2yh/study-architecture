package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/repository"
)

func (h *Handler) Checkout(c *gin.Context) {
	var req api.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	lines := make([]repository.CheckoutLine, 0, len(req.Items))
	var totalCents int64
	for _, item := range req.Items {
		snap, err := h.product.FetchProduct(c.Request.Context(), item.ProductId)
		if err != nil {
			if errors.Is(err, gateway.ErrProductNotFound) {
				_ = c.Error(middleware.Unprocessable(err.Error()))
				return
			}
			_ = c.Error(middleware.BadGateway("product service unavailable"))
			return
		}
		lines = append(lines, repository.CheckoutLine{
			ProductID:      snap.ID,
			ProductName:    snap.Name,
			UnitPriceCents: snap.UnitPriceCents,
			Quantity:       int32(item.Quantity),
		})
		totalCents += snap.UnitPriceCents * int64(item.Quantity)
	}

	order, items, err := h.repo.Checkout(c.Request.Context(), req.MemberId, "confirmed", totalCents, lines)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// 配送は同期 checkout に含めない ([[0008]])。
	if _, err := h.payment.CreatePayment(c.Request.Context(), order.ID, totalCents, req.PaymentMethod); err != nil {
		_ = c.Error(middleware.BadGateway("payment service unavailable"))
		return
	}

	c.JSON(http.StatusCreated, toAPIOrderWithItems(order, items))
}
