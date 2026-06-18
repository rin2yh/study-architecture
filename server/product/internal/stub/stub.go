package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type Repo struct {
	Products []db.ProductProduct
	Product  db.ProductProduct
	Err      error
}

func (s Repo) ListProducts(context.Context) ([]db.ProductProduct, error) {
	return s.Products, s.Err
}

func (s Repo) GetProduct(context.Context, int64) (db.ProductProduct, error) {
	return s.Product, s.Err
}

func (s Repo) CreateProduct(context.Context, db.CreateProductParams) (db.ProductProduct, error) {
	return s.Product, s.Err
}
