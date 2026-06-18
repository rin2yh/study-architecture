package stub

import (
	"context"

	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

type Repo struct {
	Products []db.ProductProduct
	Err      error
}

func (s Repo) ListProducts(context.Context) ([]db.ProductProduct, error) {
	return s.Products, s.Err
}
