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

func (r *ProductCommand) CreateProduct(ctx context.Context, arg ProductCreate) (Product, error) {
	row, err := r.q.CreateProduct(ctx, db.CreateProductParams{
		Sku:        arg.Sku,
		Name:       arg.Name,
		PriceCents: arg.PriceCents,
	})
	if err != nil {
		return Product{}, dberr.FromWrite(err)
	}
	return toProduct(row), nil
}

func (r *ProductCommand) UpdateProduct(ctx context.Context, arg ProductUpdate) (Product, error) {
	row, err := r.q.UpdateProduct(ctx, db.UpdateProductParams{
		ID:         arg.ID,
		Name:       arg.Name,
		PriceCents: arg.PriceCents,
	})
	if err != nil {
		return Product{}, dberr.FromUpdate(err)
	}
	return toProduct(row), nil
}
