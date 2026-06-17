package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-service-base-architecture/services/shipping/internal/db"
)

// ShipmentRepository は handler が依存する配送データアクセスの境界。
// interface にすることで handler のテストでスタブへ差し替えられる。
type ShipmentRepository interface {
	ListShipments(ctx context.Context) ([]db.ShippingShipment, error)
}

// Repository は pgx プール上の sqlc Querier を用いた ShipmentRepository の実装。
type Repository struct {
	q db.Querier
}

var _ ShipmentRepository = (*Repository)(nil)

// NewPool は DATABASE_URL から pgx 接続プールを生成する。
// DI では kessoku.Async でラップし、I/O を伴うこの初期化を並列化対象にしている。
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

// NewRepository は接続プールから Repository を組み立てる。
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: db.New(pool)}
}

func (r *Repository) ListShipments(ctx context.Context) ([]db.ShippingShipment, error) {
	return r.q.ListShipments(ctx)
}
