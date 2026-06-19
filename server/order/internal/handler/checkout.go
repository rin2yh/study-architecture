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

// Checkout はカートを確定する。product を参照して明細をスナップショットし、注文を
// トランザクションで作成したうえで payment / shipping を順に手配する ([[0008]])。
func (h *Handler) Checkout(c *gin.Context) {
	var req api.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}

	ctx := c.Request.Context()
	lines := make([]repository.CheckoutLine, 0, len(req.Items))
	var totalCents int64
	for _, item := range req.Items {
		snap, err := h.product.FetchProduct(ctx, item.ProductId)
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

	order, items, err := h.repo.Checkout(ctx, req.MemberId, "confirmed", totalCents, lines)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// 注文確定後に下流を手配する。別 DB のため注文 tx には含められず、ここでの失敗は
	// 注文を残したまま 502 を返す (要・整合復旧 / [[0008]] の Consequences)。
	if _, err := h.payment.CreatePayment(ctx, order.ID, totalCents, req.PaymentMethod); err != nil {
		_ = c.Error(middleware.BadGateway("payment service unavailable"))
		return
	}
	if _, err := h.shipping.CreateShipment(ctx, order.ID); err != nil {
		_ = c.Error(middleware.BadGateway("shipping service unavailable"))
		return
	}

	c.JSON(http.StatusCreated, toAPIOrderWithItems(order, items))
}
