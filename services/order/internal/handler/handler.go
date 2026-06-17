package handler

import (
	"context"

	"github.com/rin2yh/study-service-base-architecture/services/order/api"
	"github.com/rin2yh/study-service-base-architecture/services/order/internal/repository"
)

// Handler は oapi-codegen 生成の StrictServerInterface を実装する。
type Handler struct {
	repo repository.OrderRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

// New は handler を生成する。kessoku から repository.OrderRepository を注入される。
func New(repo repository.OrderRepository) *Handler {
	return &Handler{repo: repo}
}

// GetHealthz は liveness を返す（DB 非依存）。
func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

// ListOrders は注文一覧を返す。DB の行を API 表現へ詰め替える。
func (h *Handler) ListOrders(ctx context.Context, _ api.ListOrdersRequestObject) (api.ListOrdersResponseObject, error) {
	rows, err := h.repo.ListOrders(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListOrders200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, api.Order{
			Id:         r.ID,
			MemberId:   r.MemberID,
			Status:     r.Status,
			TotalCents: r.TotalCents,
			CreatedAt:  r.CreatedAt.Time,
		})
	}
	return out, nil
}
