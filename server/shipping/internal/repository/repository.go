package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-service-base-architecture/server/shipping/internal/db"
)

// interface にすることで handler のテストでスタブへ差し替えられる。
type ShipmentRepository interface {
	ListShipments(ctx context.Context) ([]db.ShippingShipment, error)
}

type Repository struct {
	q db.Querier
}

var _ ShipmentRepository = (*Repository)(nil)

// DI では kessoku.Async でラップし、I/O を伴うこの初期化を並列化対象にしている。
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: db.New(pool)}
}

func (r *Repository) ListShipments(ctx context.Context) ([]db.ShippingShipment, error) {
	return r.q.ListShipments(ctx)
}
