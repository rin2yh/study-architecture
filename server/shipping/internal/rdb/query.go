package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type ShipmentQuery struct {
	q db.Querier
}

func NewShipmentQuery(pool *pgxpool.Pool) *ShipmentQuery {
	return &ShipmentQuery{q: db.New(pool)}
}

func (r *ShipmentQuery) ListShipments(ctx context.Context) ([]db.ShippingShipment, error) {
	return r.q.ListShipments(ctx)
}

func (r *ShipmentQuery) GetShipment(ctx context.Context, id int64) (db.ShippingShipment, error) {
	row, err := r.q.GetShipment(ctx, id)
	if err != nil {
		return db.ShippingShipment{}, dberr.FromRead(err)
	}
	return row, nil
}
