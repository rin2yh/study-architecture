package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type ProductRepository interface {
	ListProducts(ctx context.Context) ([]db.ProductProduct, error)
	GetProduct(ctx context.Context, id int64) (db.ProductProduct, error)
	CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.ProductProduct, error)
}

type Repository struct {
	q db.Querier
}

var _ ProductRepository = (*Repository)(nil)

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

func (r *Repository) ListProducts(ctx context.Context) ([]db.ProductProduct, error) {
	return r.q.ListProducts(ctx)
}

func (r *Repository) GetProduct(ctx context.Context, id int64) (db.ProductProduct, error) {
	row, err := r.q.GetProduct(ctx, id)
	if err != nil {
		return db.ProductProduct{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *Repository) CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.ProductProduct, error) {
	row, err := r.q.CreateProduct(ctx, arg)
	if err != nil {
		return db.ProductProduct{}, dberr.FromWrite(err)
	}
	return row, nil
}
