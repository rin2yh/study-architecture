package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type ShipmentStub struct {
	Shipments []db.ShippingShipment
	Shipment  db.ShippingShipment
	Err       error
}

func (s ShipmentStub) ListShipments(context.Context) ([]db.ShippingShipment, error) {
	return s.Shipments, s.Err
}

func (s ShipmentStub) GetShipment(context.Context, int64) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}

func (s ShipmentStub) CreateShipment(context.Context, db.CreateShipmentParams) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}

func (s ShipmentStub) UpdateShipment(context.Context, db.UpdateShipmentParams) (db.ShippingShipment, error) {
	return s.Shipment, s.Err
}
