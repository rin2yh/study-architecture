package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/shipping/api"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type Query interface {
	ListShipments(ctx context.Context) ([]db.ShippingShipment, error)
	GetShipment(ctx context.Context, id int64) (db.ShippingShipment, error)
}

type Command interface {
	CreateShipment(ctx context.Context, arg db.CreateShipmentParams) (db.ShippingShipment, error)
	UpdateShipment(ctx context.Context, arg db.UpdateShipmentParams) (db.ShippingShipment, error)
}

type Handler struct {
	query   Query
	command Command
}

var _ api.ServerInterface = (*Handler)(nil)

func New(query Query, command Command) *Handler {
	return &Handler{query: query, command: command}
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
