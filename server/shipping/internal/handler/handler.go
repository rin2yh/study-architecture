package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
