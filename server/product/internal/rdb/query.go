package rdb

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type ProductQuery struct {
	q db.Querier
}

func NewProductQuery(pool *pgxpool.Pool) *ProductQuery {
	return &ProductQuery{q: db.New(pool)}
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
