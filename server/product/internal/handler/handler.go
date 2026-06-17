package handler

import (
	"context"

	"github.com/rin2yh/study-architecture/server/product/api"
	"github.com/rin2yh/study-architecture/server/product/internal/repository"
)

type Handler struct {
	repo repository.ProductRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(repo repository.ProductRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

func (h *Handler) ListProducts(ctx context.Context, _ api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	rows, err := h.repo.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListProducts200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, api.Product{
			Id:         r.ID,
			Sku:        r.Sku,
			Name:       r.Name,
			PriceCents: r.PriceCents,
			CreatedAt:  r.CreatedAt.Time,
		})
	}
	return out, nil
}
