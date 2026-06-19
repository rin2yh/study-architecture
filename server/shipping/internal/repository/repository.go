package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type ShipmentQuery struct {
	q db.Querier
}

type ShipmentCommand struct {
	q db.Querier
}

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

func NewShipmentQuery(pool *pgxpool.Pool) *ShipmentQuery {
	return &ShipmentQuery{q: db.New(pool)}
}

func NewShipmentCommand(pool *pgxpool.Pool) *ShipmentCommand {
	return &ShipmentCommand{q: db.New(pool)}
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

func (r *ShipmentCommand) CreateShipment(ctx context.Context, arg db.CreateShipmentParams) (db.ShippingShipment, error) {
	return r.q.CreateShipment(ctx, arg)
}

func (r *ShipmentCommand) UpdateShipment(ctx context.Context, arg db.UpdateShipmentParams) (db.ShippingShipment, error) {
	row, err := r.q.UpdateShipment(ctx, arg)
	if err != nil {
		return db.ShippingShipment{}, dberr.FromUpdate(err)
	}
	return row, nil
}
