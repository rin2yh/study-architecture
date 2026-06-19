package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/shipping/api"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

func (h *Handler) CreateShipment(c *gin.Context) {
	var req api.CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.repo.CreateShipment(c.Request.Context(), db.CreateShipmentParams{
		OrderID:    req.OrderId,
		Carrier:    req.Carrier,
		TrackingNo: req.TrackingNo,
		Status:     req.Status,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIShipment(row))
}

func (h *Handler) UpdateShipment(c *gin.Context, id api.IdPath) {
	var req api.UpdateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	row, err := h.repo.UpdateShipment(c.Request.Context(), db.UpdateShipmentParams{
		ID:     id,
		Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("shipment not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIShipment(row))
}
