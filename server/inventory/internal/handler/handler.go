package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/inventory/api"
	"github.com/rin2yh/study-architecture/server/inventory/internal/rdb"
)

type Query interface {
	Available(ctx context.Context, productID int64) (int64, error)
}

type Command interface {
	StockIn(ctx context.Context, productID int64, quantity int32) (rdb.StockIn, error)
	Reserve(ctx context.Context, orderID int64, lines []rdb.ReserveLine) error
	ReleaseReservationsByOrder(ctx context.Context, orderID int64) error
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

func toAPIStockIn(r rdb.StockIn) api.StockIn {
	return api.StockIn{
		Id:        r.ID,
		ProductId: r.ProductID,
		Quantity:  int(r.Quantity),
	}
}
