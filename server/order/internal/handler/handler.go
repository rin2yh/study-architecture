package handler

import (
	"context"

	"github.com/rin2yh/study-service-base-architecture/server/order/api"
	"github.com/rin2yh/study-service-base-architecture/server/order/internal/repository"
)

type Handler struct {
	repo repository.OrderRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(repo repository.OrderRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

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
