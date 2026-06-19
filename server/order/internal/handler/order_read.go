package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/order/api"
)

func (h *Handler) ListOrders(c *gin.Context) {
	rows, err := h.repo.ListOrders(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]api.Order, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPIOrder(r))
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) GetOrder(c *gin.Context, id api.IdPath) {
	row, err := h.repo.GetOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("order not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIOrder(row))
}
