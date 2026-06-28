package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type ShipmentCommand struct {
	q db.Querier
}

func NewShipmentCommand(pool *pgxpool.Pool) *ShipmentCommand {
	return &ShipmentCommand{q: db.New(pool)}
}

func (r *ShipmentCommand) CreateShipment(ctx context.Context, arg db.CreateShipmentParams) (db.ShippingShipment, error) {
	return r.q.CreateShipment(ctx, arg)
}

func (r *ShipmentCommand) CreateShipmentForOrder(ctx context.Context, orderID int64, dest paymentevent.Destination) (db.ShippingShipment, error) {
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

func (r *ShipmentCommand) UpdateShipment(ctx context.Context, arg db.UpdateShipmentParams) (db.ShippingShipment, error) {
	row, err := r.q.UpdateShipment(ctx, arg)
	if err != nil {
		return db.ShippingShipment{}, dberr.FromUpdate(err)
	}
	return row, nil
}
