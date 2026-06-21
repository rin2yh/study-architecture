package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type ProductStub struct {
	Products []db.ProductProduct
	Product  db.ProductProduct
	Err      error
}

func (s ProductStub) ListProducts(context.Context) ([]db.ProductProduct, error) {
	return s.Products, s.Err
}

func (s ProductStub) GetProduct(context.Context, int64) (db.ProductProduct, error) {
	return s.Product, s.Err
}

func (s ProductStub) CreateProduct(context.Context, db.CreateProductParams) (db.ProductProduct, error) {
	return s.Product, s.Err
}

func (s ProductStub) UpdateProduct(context.Context, db.UpdateProductParams) (db.ProductProduct, error) {
	return s.Product, s.Err
}
