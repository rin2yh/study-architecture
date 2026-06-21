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

type readHandler struct {
	query Query
}

type writeHandler struct {
	command Command
}

// Handler は readHandler / writeHandler を束ねて単一の api.ServerInterface を満たす。
// 各サブハンドラは Query / Command の片側にしか依存しないため、読み書きの依存が型で分離される。
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
