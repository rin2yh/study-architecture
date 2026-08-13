package rdb

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
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

func (r *ShipmentCommand) CreateShipmentForOrder(ctx context.Context, orderID order.ID, dest gateway.Destination) (db.ShippingShipment, error) {
	id, err := toInt64(orderID.String())
	if err != nil {
		return db.ShippingShipment{}, err
	}
	row, err := r.q.CreateShipmentForOrder(ctx, db.CreateShipmentForOrderParams{
		OrderID:        id,
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

func (r *ShipmentCommand) CancelShipmentForOrder(ctx context.Context, orderID order.ID) error {
	id, err := toInt64(orderID.String())
	if err != nil {
		return err
	}
	return r.q.CancelShipmentForOrder(ctx, id)
}

// ID を数値へ戻すのはここだけ。列と生成コードが bigint / int64 なのはこの層の事情で、
// ドメインの order.ID は表現を持たない。
func toInt64(raw string) (int64, error) {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", raw, err)
	}
	return v, nil
}
