package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/shipping/api"
	"github.com/rin2yh/study-architecture/server/shipping/internal/rdb"
)

type Query interface {
	ListShipments(ctx context.Context) ([]rdb.Shipment, error)
	GetShipment(ctx context.Context, id int64) (rdb.Shipment, error)
}

type Command interface {
	CreateShipment(ctx context.Context, arg rdb.ShipmentCreate) (rdb.Shipment, error)
	UpdateShipment(ctx context.Context, arg rdb.ShipmentUpdate) (rdb.Shipment, error)
}

type readHandler struct {
	query Query
}

type writeHandler struct {
	command Command
}

type Handler struct {
	*readHandler
	*writeHandler
}

var _ api.ServerInterface = (*Handler)(nil)

func New(query Query, command Command) *Handler {
	return &Handler{
		readHandler:  &readHandler{query: query},
		writeHandler: &writeHandler{command: command},
	}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func toAPIShipment(r rdb.Shipment) api.Shipment {
	return api.Shipment{
		Id:         r.ID,
		OrderId:    r.OrderID,
		Carrier:    r.Carrier,
		TrackingNo: r.TrackingNo,
		Status:     r.Status,
		Destination: api.Destination{
			Recipient:  r.ShipRecipient,
			PostalCode: r.ShipPostalCode,
			Prefecture: r.ShipPrefecture,
			City:       r.ShipCity,
			Line1:      r.ShipLine1,
		},
		CreatedAt: r.CreatedAt,
	}
}
