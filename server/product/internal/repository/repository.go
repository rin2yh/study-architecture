package repository

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type ProductQuery struct {
	q db.Querier
}

type ProductCommand struct {
	q db.Querier
}

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return pgxpool.New(ctx, dsn)
}

func NewProductQuery(pool *pgxpool.Pool) *ProductQuery {
	return &ProductQuery{q: db.New(pool)}
}

func NewProductCommand(pool *pgxpool.Pool) *ProductCommand {
	return &ProductCommand{q: db.New(pool)}
}

func (r *ProductQuery) ListProducts(ctx context.Context) ([]db.ProductProduct, error) {
	return r.q.ListProducts(ctx)
}

func (r *ProductQuery) GetProduct(ctx context.Context, id int64) (db.ProductProduct, error) {
	row, err := r.q.GetProduct(ctx, id)
	if err != nil {
		return db.ProductProduct{}, dberr.FromRead(err)
	}
	return row, nil
}

func (r *ProductCommand) CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.ProductProduct, error) {
	row, err := r.q.CreateProduct(ctx, arg)
	if err != nil {
		return db.ProductProduct{}, dberr.FromWrite(err)
	}
	return row, nil
}

func (r *ProductCommand) UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.ProductProduct, error) {
	row, err := r.q.UpdateProduct(ctx, arg)
	if err != nil {
		return db.ProductProduct{}, dberr.FromUpdate(err)
	}
	return row, nil
}
