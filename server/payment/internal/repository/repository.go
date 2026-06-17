package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

// interface にすることで handler のテストでスタブへ差し替えられる。
type PaymentRepository interface {
	ListPayments(ctx context.Context) ([]db.PaymentPayment, error)
}

type Repository struct {
	q db.Querier
}

var _ PaymentRepository = (*Repository)(nil)

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

func (r *Repository) ListPayments(ctx context.Context) ([]db.PaymentPayment, error) {
	return r.q.ListPayments(ctx)
}
