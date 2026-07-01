package rdb

import (
	"time"

	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

// port を sqlc 生成型から切り離す (ADR-[[202607011200]])。pgtype を標準型へ寄せ、schema 変更を
// rdb のマッピングに閉じ込める。
type Order struct {
	ID                 int64
	MemberID           int64
	Status             string
	TotalCents         int64
	CreatedAt          time.Time
	ShippingRecipient  string
	ShippingPostalCode string
	ShippingPrefecture string
	ShippingCity       string
	ShippingLine1      string
}

type OrderItem struct {
	ID             int64
	OrderID        int64
	ProductID      int64
	ProductName    string
	UnitPriceCents int64
	Quantity       int32
	CreatedAt      time.Time
}

type OrderCreate struct {
	MemberID   int64
	Status     string
	TotalCents int64
}

type OrderUpdate struct {
	ID     int64
	Status string
}

func toOrder(r db.OrderOrder) Order {
	return Order{
		ID:                 r.ID,
		MemberID:           r.MemberID,
		Status:             r.Status,
		TotalCents:         r.TotalCents,
		CreatedAt:          r.CreatedAt.Time,
		ShippingRecipient:  r.ShippingRecipient,
		ShippingPostalCode: r.ShippingPostalCode,
		ShippingPrefecture: r.ShippingPrefecture,
		ShippingCity:       r.ShippingCity,
		ShippingLine1:      r.ShippingLine1,
	}
}

func toOrders(rows []db.OrderOrder) []Order {
	out := make([]Order, 0, len(rows))
	for _, r := range rows {
		out = append(out, toOrder(r))
	}
	return out
}

func toOrderItem(r db.OrderOrderItem) OrderItem {
	return OrderItem{
		ID:             r.ID,
		OrderID:        r.OrderID,
		ProductID:      r.ProductID,
		ProductName:    r.ProductName,
		UnitPriceCents: r.UnitPriceCents,
		Quantity:       r.Quantity,
		CreatedAt:      r.CreatedAt.Time,
	}
}

func toOrderItems(rows []db.OrderOrderItem) []OrderItem {
	out := make([]OrderItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, toOrderItem(r))
	}
	return out
}
