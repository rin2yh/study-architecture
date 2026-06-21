package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type RDB struct {
	Shipments []db.ShippingShipment
	Shipment  db.ShippingShipment
	Err       error
}

func (s RDB) ListShipments(context.Context) ([]db.ShippingShipment, error) {
	return s.Shipments, s.Err
}

func (s RDB) GetShipment(context.Context, int64) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}

func (s RDB) CreateShipment(context.Context, db.CreateShipmentParams) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}

func (s RDB) UpdateShipment(context.Context, db.UpdateShipmentParams) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}
