package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/product/internal/rdb"
)

type ProductStub struct {
	Products []rdb.Product
	Product  rdb.Product
	Err      error
}

func (s ProductStub) ListProducts(context.Context) ([]rdb.Product, error) {
	return s.Products, s.Err
}

func (s ProductStub) GetProduct(context.Context, int64) (rdb.Product, error) {
	return s.Product, s.Err
}

func (s ProductStub) CreateProduct(context.Context, rdb.ProductCreate) (rdb.Product, error) {
	return s.Product, s.Err
}

func (s ProductStub) UpdateProduct(context.Context, rdb.ProductUpdate) (rdb.Product, error) {
	return s.Product, s.Err
}
