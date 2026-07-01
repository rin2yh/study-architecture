package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/shipping/internal/rdb"
)

type ShipmentStub struct {
	Shipments []rdb.Shipment
	Shipment  rdb.Shipment
	Err       error
}

func (s ShipmentStub) ListShipments(context.Context) ([]rdb.Shipment, error) {
	return s.Shipments, s.Err
}

func (s ShipmentStub) GetShipment(context.Context, int64) (rdb.Shipment, error) {
	return s.Shipment, s.Err
}

func (s ShipmentStub) CreateShipment(context.Context, rdb.ShipmentCreate) (rdb.Shipment, error) {
	return s.Shipment, s.Err
}

func (s ShipmentStub) UpdateShipment(context.Context, rdb.ShipmentUpdate) (rdb.Shipment, error) {
	return s.Shipment, s.Err
}
