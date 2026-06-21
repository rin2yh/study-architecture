package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/shipping/api"
)

func (h *readHandler) ListShipments(c *gin.Context) {
	rows, err := h.query.ListShipments(c.Request.Context())
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

func (h *readHandler) GetShipment(c *gin.Context, id api.IdPath) {
	row, err := h.query.GetShipment(c.Request.Context(), id)
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
