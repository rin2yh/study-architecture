package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type Query interface {
	ListProducts(ctx context.Context) ([]db.ProductProduct, error)
	GetProduct(ctx context.Context, id int64) (db.ProductProduct, error)
}

type Command interface {
	CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.ProductProduct, error)
	UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.ProductProduct, error)
}

type readHandler struct {
	query Query
}

type writeHandler struct {
	command Command
}

// Handler は readHandler / writeHandler を束ねて単一の api.ServerInterface を満たす。
// 各サブハンドラは Query / Command の片側にしか依存しないため、読み書きの依存が型で分離される。
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

func toAPIProduct(r db.ProductProduct) api.Product {
	return api.Product{
		Id:         r.ID,
		Sku:        r.Sku,
		Name:       r.Name,
		PriceCents: r.PriceCents,
		CreatedAt:  r.CreatedAt.Time,
	}
}
