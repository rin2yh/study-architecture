package handler

import (
	"context"

	"github.com/rin2yh/study-service-base-architecture/server/shipping/api"
	"github.com/rin2yh/study-service-base-architecture/server/shipping/internal/repository"
)

type Handler struct {
	repo repository.ShipmentRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(repo repository.ShipmentRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

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
