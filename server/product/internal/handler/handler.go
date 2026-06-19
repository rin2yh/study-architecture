package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/middleware"
	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
	"github.com/rin2yh/study-architecture/server/product/internal/repository"
)

type Handler struct {
	repo repository.ProductRepository
}

var _ api.ServerInterface = (*Handler)(nil)

func New(repo repository.ProductRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) ListProducts(c *gin.Context) {
	rows, err := h.repo.ListProducts(c.Request.Context())
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

func (h *Handler) GetProduct(c *gin.Context, id api.IdPath) {
	row, err := h.repo.GetProduct(c.Request.Context(), id)
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
	row, err := h.repo.CreateProduct(c.Request.Context(), db.CreateProductParams{
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
	row, err := h.repo.UpdateProduct(c.Request.Context(), db.UpdateProductParams{
		ID:         id,
		Sku:        req.Sku,
		Name:       req.Name,
		PriceCents: req.PriceCents,
	})
	if err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			_ = c.Error(middleware.NotFound("product not found"))
			return
		}
		if errors.Is(err, dberr.ErrConflict) {
			_ = c.Error(middleware.Conflict("product with this sku already exists"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toAPIProduct(row))
}

func toAPIProduct(r db.ProductProduct) api.Product {
	return api.Product{
		Id:         r.ID,
		Sku:        r.Sku,
		Name:       r.Name,
		PriceCents: r.PriceCents,
		CreatedAt:  r.CreatedAt.Time,
	}
}
