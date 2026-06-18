package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type Repo struct {
	Shipments []db.ShippingShipment
	Err       error
}

func (s Repo) ListShipments(context.Context) ([]db.ShippingShipment, error) {
	return s.Shipments, s.Err
}
