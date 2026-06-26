package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/inventory/api"
	"github.com/rin2yh/study-architecture/server/inventory/internal/rdb"
)

func (h *writeHandler) StockIn(c *gin.Context) {
	var req api.StockInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.command.StockIn(c.Request.Context(), req.ProductId, int32(req.Quantity))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIMovement(row))
}

func (h *writeHandler) Reserve(c *gin.Context) {
	var req api.ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	lines := make([]rdb.ReserveLine, 0, len(req.Lines))
	for _, l := range req.Lines {
		lines = append(lines, rdb.ReserveLine{ProductID: l.ProductId, Quantity: int32(l.Quantity)})
	}
	ttl := int32(defaultReserveTTLSeconds)
	if req.TtlSeconds != nil && *req.TtlSeconds > 0 {
		ttl = int32(*req.TtlSeconds)
	}
	if err := h.command.Reserve(c.Request.Context(), req.OrderId, lines, ttl); err != nil {
		if errors.Is(err, rdb.ErrInsufficientStock) {
			_ = c.Error(middleware.Conflict("insufficient stock"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, api.ReservationResult{OrderId: req.OrderId})
}

func (h *writeHandler) ReleaseReservation(c *gin.Context, orderId api.OrderIdPath) {
	if err := h.command.ReleaseReservationsByOrder(c.Request.Context(), orderId); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
