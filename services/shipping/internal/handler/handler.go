package handler

import (
	"context"

	"github.com/rin2yh/study-service-base-architecture/services/shipping/api"
	"github.com/rin2yh/study-service-base-architecture/services/shipping/internal/repository"
)

// Handler は oapi-codegen 生成の StrictServerInterface を実装する。
type Handler struct {
	repo repository.ShipmentRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

// New は handler を生成する。kessoku から repository.ShipmentRepository を注入される。
func New(repo repository.ShipmentRepository) *Handler {
	return &Handler{repo: repo}
}

// GetHealthz は liveness を返す（DB 非依存）。
func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

// ListShipments は配送一覧を返す。DB の行を API 表現へ詰め替える。
func (h *Handler) ListShipments(ctx context.Context, _ api.ListShipmentsRequestObject) (api.ListShipmentsResponseObject, error) {
	rows, err := h.repo.ListShipments(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListShipments200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, api.Shipment{
			Id:         r.ID,
			OrderId:    r.OrderID,
			Carrier:    r.Carrier,
			TrackingNo: r.TrackingNo,
			Status:     r.Status,
			CreatedAt:  r.CreatedAt.Time,
		})
	}
	return out, nil
}
