package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/order/api"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

func (h *Handler) CreateOrder(c *gin.Context) {
	var req api.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.TotalCents < 0 {
		_ = c.Error(middleware.Unprocessable("totalCents must not be negative"))
		return
	}
	row, err := h.repo.CreateOrder(c.Request.Context(), db.CreateOrderParams{
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

func (h *Handler) UpdateOrder(c *gin.Context, id api.IdPath) {
	var req api.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.repo.UpdateOrder(c.Request.Context(), db.UpdateOrderParams{
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
