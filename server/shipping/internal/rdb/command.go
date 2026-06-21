package rdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
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

func (r *ShipmentCommand) CreateShipmentForOrder(ctx context.Context, orderID int64) (db.ShippingShipment, error) {
	row, err := r.q.CreateShipmentForOrder(ctx, orderID)
	// ON CONFLICT DO NOTHING は重複時に行を返さず no rows になる。これは失敗ではなく
	// 「既に手配済み」なので、再配送を冪等に握れるよう ErrConflict に正規化する。
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ShippingShipment{}, dberr.ErrConflict
	}
	return row, err
}

func (r *ShipmentCommand) UpdateShipment(ctx context.Context, arg db.UpdateShipmentParams) (db.ShippingShipment, error) {
	row, err := r.q.UpdateShipment(ctx, arg)
	if err != nil {
		return db.ShippingShipment{}, dberr.FromUpdate(err)
	}
	return row, nil
}
