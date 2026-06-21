package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/product/api"
)

func (h *readHandler) ListProducts(c *gin.Context) {
	rows, err := h.query.ListProducts(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]api.Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAPIProduct(r))
	}
	c.JSON(http.StatusOK, out)
}

func (h *readHandler) GetProduct(c *gin.Context, id api.IdPath) {
	row, err := h.query.GetProduct(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("product not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIProduct(row))
}
