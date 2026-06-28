package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/inventory/api"
)

func (h *readHandler) GetAvailability(c *gin.Context, productId api.ProductIdPath) {
	available, err := h.query.Available(c.Request.Context(), productId)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, api.Availability{ProductId: productId, Available: available})
}
