package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type Repo struct {
	Shipments []db.ShippingShipment
	Shipment  db.ShippingShipment
	Err       error
}

func (s Repo) ListShipments(context.Context) ([]db.ShippingShipment, error) {
	return s.Shipments, s.Err
}

func (s Repo) GetShipment(context.Context, int64) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}

func (s Repo) CreateShipment(context.Context, db.CreateShipmentParams) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}
