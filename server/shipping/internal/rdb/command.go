package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
)

type ShipmentCommand struct {
	q db.Querier
}

func NewShipmentCommand(pool *pgxpool.Pool) *ShipmentCommand {
	return &ShipmentCommand{q: db.New(pool)}
}

func (r *ShipmentCommand) CreateShipment(ctx context.Context, arg ShipmentCreate) (Shipment, error) {
	row, err := r.q.CreateShipment(ctx, db.CreateShipmentParams{
		OrderID:    arg.OrderID,
		Carrier:    arg.Carrier,
		TrackingNo: arg.TrackingNo,
		Status:     arg.Status,
	})
	if err != nil {
		return Shipment{}, err
	}
	return toShipment(row), nil
}

func (r *ShipmentCommand) CreateShipmentForOrder(ctx context.Context, orderID int64, dest gateway.Destination) (db.ShippingShipment, error) {
	row, err := r.q.CreateShipmentForOrder(ctx, db.CreateShipmentForOrderParams{
		OrderID:        orderID,
		ShipRecipient:  dest.Recipient,
		ShipPostalCode: dest.PostalCode,
		ShipPrefecture: dest.Prefecture,
		ShipCity:       dest.City,
		ShipLine1:      dest.Line1,
	})
	return row, dberr.FromInsertSkipped(err)
}

func (r *ShipmentCommand) UpdateShipment(ctx context.Context, arg ShipmentUpdate) (Shipment, error) {
	row, err := r.q.UpdateShipment(ctx, db.UpdateShipmentParams{ID: arg.ID, Status: arg.Status})
	if err != nil {
		return Shipment{}, dberr.FromUpdate(err)
	}
	return toShipment(row), nil
}

func (r *ShipmentCommand) CancelShipmentForOrder(ctx context.Context, orderID int64) error {
	return r.q.CancelShipmentForOrder(ctx, orderID)
}
