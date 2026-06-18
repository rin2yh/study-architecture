package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/shipping/api"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/repository"
)

type Handler struct {
	repo repository.ShipmentRepository
}

var _ api.ServerInterface = (*Handler)(nil)

func New(repo repository.ShipmentRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ListShipments(c *gin.Context) {
	rows, err := h.repo.ListShipments(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]api.Shipment, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPIShipment(r))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) GetShipment(c *gin.Context, id api.IdPath) {
	row, err := h.repo.GetShipment(c.Request.Context(), id)
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

func toAPIShipment(r db.ShippingShipment) api.Shipment {
	return api.Shipment{
		Id:         r.ID,
		OrderId:    r.OrderID,
		Carrier:    r.Carrier,
		TrackingNo: r.TrackingNo,
		Status:     r.Status,
		CreatedAt:  r.CreatedAt.Time,
	}
}
