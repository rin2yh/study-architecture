package handler

import (
	"context"

	"github.com/rin2yh/study-service-base-architecture/services/product/api"
	"github.com/rin2yh/study-service-base-architecture/services/product/internal/repository"
)

// Handler は oapi-codegen 生成の StrictServerInterface を実装する。
type Handler struct {
	repo repository.ProductRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

// New は handler を生成する。kessoku から repository.ProductRepository を注入される。
func New(repo repository.ProductRepository) *Handler {
	return &Handler{repo: repo}
}

// GetHealthz は liveness を返す（DB 非依存）。
func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

// ListProducts は商品一覧を返す。DB の行を API 表現へ詰め替える。
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
