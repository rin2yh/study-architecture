package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
	"github.com/rin2yh/study-architecture/server/order/internal/gateway"
	"github.com/rin2yh/study-architecture/server/order/internal/rdb"
)

func (h *writeHandler) CreateOrder(c *gin.Context) {
	var req api.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.TotalCents < 0 {
		_ = c.Error(middleware.Unprocessable("totalCents must not be negative"))
		return
	}
	row, err := h.command.CreateOrder(c.Request.Context(), db.CreateOrderParams{
		MemberID:   req.MemberId,
		Status:     req.Status,
		TotalCents: req.TotalCents,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIOrder(row))
}

func (h *writeHandler) UpdateOrder(c *gin.Context, id api.IdPath) {
	var req api.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.command.UpdateOrder(c.Request.Context(), db.UpdateOrderParams{
		ID:     id,
		Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("order not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIOrder(row))
}

func (h *writeHandler) Checkout(c *gin.Context) {
	var req api.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	lines := make([]rdb.CheckoutLine, 0, len(req.Items))
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
		lines = append(lines, rdb.CheckoutLine{
			ProductID:      snap.ID,
			ProductName:    snap.Name,
			UnitPriceCents: snap.UnitPriceCents,
			Quantity:       int32(item.Quantity),
		})
		totalCents += snap.UnitPriceCents * int64(item.Quantity)
	}

	order, items, err := h.command.Checkout(c.Request.Context(), req.MemberId, "confirmed", totalCents, lines)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// [[0008]]
	if _, err := h.payment.CreatePayment(c.Request.Context(), order.ID, totalCents, req.PaymentMethod); err != nil {
		_ = c.Error(middleware.BadGateway("payment service unavailable"))
		return
	}

	c.JSON(http.StatusCreated, toAPIOrderWithItems(order, items))
}
