package handler

import (
	"context"

	"github.com/rin2yh/study-architecture/server/payment/api"
	"github.com/rin2yh/study-architecture/server/payment/internal/repository"
)

type Handler struct {
	repo repository.PaymentRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(repo repository.PaymentRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

func (h *Handler) ListPayments(ctx context.Context, _ api.ListPaymentsRequestObject) (api.ListPaymentsResponseObject, error) {
	rows, err := h.repo.ListPayments(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListPayments200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, api.Payment{
			Id:          r.ID,
			OrderId:     r.OrderID,
			AmountCents: r.AmountCents,
			Method:      r.Method,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt.Time,
		})
	}
	return out, nil
}
