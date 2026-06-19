package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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

func toAPIProduct(r db.ProductProduct) api.Product {
	return api.Product{
		Id:         r.ID,
		Sku:        r.Sku,
		Name:       r.Name,
		PriceCents: r.PriceCents,
		CreatedAt:  r.CreatedAt.Time,
	}
}
