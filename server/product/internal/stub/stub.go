package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type RDB struct {
	Products []db.ProductProduct
	Product  db.ProductProduct
	Err      error
}

func (s RDB) ListProducts(context.Context) ([]db.ProductProduct, error) {
	return s.Products, s.Err
}

func (s RDB) GetProduct(context.Context, int64) (db.ProductProduct, error) {
	return s.Product, s.Err
}

func (s RDB) CreateProduct(context.Context, db.CreateProductParams) (db.ProductProduct, error) {
	return s.Product, s.Err
}

func (s RDB) UpdateProduct(context.Context, db.UpdateProductParams) (db.ProductProduct, error) {
	return s.Product, s.Err
}
