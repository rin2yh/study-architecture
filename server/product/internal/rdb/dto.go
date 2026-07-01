package rdb

import (
	"time"

	"github.com/rin2yh/study-architecture/server/product/internal/db"
)

// port を sqlc 生成型から切り離す (ADR-[[202607011200]])。pgtype は標準型へ寄せる。
type Product struct {
	ID         int64
	Sku        string
	Name       string
	PriceCents int64
	CreatedAt  time.Time
}

type ProductCreate struct {
	Sku        string
	Name       string
	PriceCents int64
}

type ProductUpdate struct {
	ID         int64
	Name       string
	PriceCents int64
}

func toProduct(r db.ProductProduct) Product {
	return Product{
		ID:         r.ID,
		Sku:        r.Sku,
		Name:       r.Name,
		PriceCents: r.PriceCents,
		CreatedAt:  r.CreatedAt.Time,
	}
}

func toProducts(rows []db.ProductProduct) []Product {
	out := make([]Product, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProduct(r))
	}
	return out
}
