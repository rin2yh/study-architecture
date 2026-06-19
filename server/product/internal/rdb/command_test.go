package rdb

import (
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

func TestProductCommandCreateProduct(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewProductCommand(pool)
	seedProducts(t, pool, db.ProductProduct{Sku: "SKU-EXIST", Name: "既存", PriceCents: 1000})

	t.Run("正常系 作成行を返す", func(t *testing.T) {
		got, err := r.CreateProduct(t.Context(), db.CreateProductParams{Sku: "SKU-NEW", Name: "新規商品", PriceCents: 2980})
		if err != nil {
			t.Fatalf("CreateProduct: %v", err)
		}
		if got.ID == 0 || got.Sku != "SKU-NEW" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 sku 重複は ErrConflict", func(t *testing.T) {
		if _, err := r.CreateProduct(t.Context(), db.CreateProductParams{Sku: "SKU-EXIST", Name: "重複", PriceCents: 100}); !errors.Is(err, dberr.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})
}

func TestProductCommandUpdateProduct(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewProductCommand(pool)
	seedProducts(t, pool, db.ProductProduct{Sku: "SKU-1", Name: "商品1", PriceCents: 1980})

	t.Run("正常系 既存行を更新して返す (sku は不変)", func(t *testing.T) {
		got, err := r.UpdateProduct(t.Context(), db.UpdateProductParams{ID: 1, Name: "商品1更新", PriceCents: 999})
		if err != nil {
			t.Fatalf("UpdateProduct: %v", err)
		}
		if got.ID != 1 || got.Name != "商品1更新" || got.PriceCents != 999 || got.Sku != "SKU-1" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateProduct(t.Context(), db.UpdateProductParams{ID: 9999, Name: "x", PriceCents: 1}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
