package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

func (h *Handler) CreateProduct(c *gin.Context) {
	var req api.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.PriceCents < 0 {
		_ = c.Error(middleware.Unprocessable("priceCents must not be negative"))
		return
	}
	row, err := h.command.CreateProduct(c.Request.Context(), db.CreateProductParams{
		Sku:        req.Sku,
		Name:       req.Name,
		PriceCents: req.PriceCents,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrConflict) {
			_ = c.Error(middleware.Conflict("product with this sku already exists"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toAPIProduct(row))
}

func (h *Handler) UpdateProduct(c *gin.Context, id api.IdPath) {
	var req api.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypeBind)
		return
	}
	if req.PriceCents < 0 {
		_ = c.Error(middleware.Unprocessable("priceCents must not be negative"))
		return
	}
	row, err := h.command.UpdateProduct(c.Request.Context(), db.UpdateProductParams{
		ID:         id,
		Name:       req.Name,
		PriceCents: req.PriceCents,
	})
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
