package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/rdb"
)

type Query interface {
	ListProducts(ctx context.Context) ([]rdb.Product, error)
	GetProduct(ctx context.Context, id int64) (rdb.Product, error)
}

type Command interface {
	CreateProduct(ctx context.Context, arg rdb.ProductCreate) (rdb.Product, error)
	UpdateProduct(ctx context.Context, arg rdb.ProductUpdate) (rdb.Product, error)
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

func toAPIProduct(r rdb.Product) api.Product {
	return api.Product{
		Id:         r.ID,
		Sku:        r.Sku,
		Name:       r.Name,
		PriceCents: r.PriceCents,
		CreatedAt:  r.CreatedAt,
	}
}
