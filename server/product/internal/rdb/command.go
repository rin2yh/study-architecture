package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type ProductCommand struct {
	q db.Querier
}

func NewProductCommand(pool *pgxpool.Pool) *ProductCommand {
	return &ProductCommand{q: db.New(pool)}
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
