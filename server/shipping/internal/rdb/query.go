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

func (r *ShipmentQuery) ListShipments(ctx context.Context) ([]Shipment, error) {
	rows, err := r.q.ListShipments(ctx)
	if err != nil {
		return nil, err
	}
	return toShipments(rows), nil
}

func (r *ShipmentQuery) GetShipment(ctx context.Context, id int64) (Shipment, error) {
	row, err := r.q.GetShipment(ctx, id)
	if err != nil {
		return Shipment{}, dberr.FromRead(err)
	}
	return toShipment(row), nil
}
