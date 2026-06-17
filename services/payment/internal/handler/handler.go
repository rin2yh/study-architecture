package handler

import (
	"context"

	"github.com/rin2yh/study-service-base-architecture/services/payment/api"
	"github.com/rin2yh/study-service-base-architecture/services/payment/internal/repository"
)

// Handler は oapi-codegen 生成の StrictServerInterface を実装する。
type Handler struct {
	repo repository.PaymentRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

// New は handler を生成する。kessoku から repository.PaymentRepository を注入される。
func New(repo repository.PaymentRepository) *Handler {
	return &Handler{repo: repo}
}

// GetHealthz は liveness を返す（DB 非依存）。
func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

// ListPayments は決済一覧を返す。DB の行を API 表現へ詰め替える。
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
