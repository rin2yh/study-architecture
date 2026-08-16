package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/inventory/api"
	"github.com/rin2yh/study-architecture/server/inventory/internal/db"
)

func (h *readHandler) GetAvailability(c *gin.Context, productId api.ProductIdPath) {
	available, err := h.query.Available(c.Request.Context(), productId)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, api.Availability{ProductId: productId, Available: available})
}

func (h *readHandler) ListReservationsByOrder(c *gin.Context, orderId api.OrderIdPath) {
	rows, err := h.query.ReservationsByOrder(c.Request.Context(), orderId)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIReservations(rows))
}

func toAPIReservations(rows []db.ListReservationsByOrderRow) []api.Reservation {
	out := make([]api.Reservation, 0, len(rows))
	for _, r := range rows {
		out = append(out, api.Reservation{
			Id:        r.ID,
			ProductId: r.ProductID,
			Quantity:  int(r.Quantity),
			State:     api.ReservationState(r.State),
		})
	}
	return out
}
