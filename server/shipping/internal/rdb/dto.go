package rdb

import (
	"time"

	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

// port を sqlc 生成型から切り離す (ADR-[[202607011200]])。pgtype は標準型へ寄せ、
// 宛先列は toAPIShipment が読む形のまま平坦に持つ。
type Shipment struct {
	ID             int64
	OrderID        int64
	Carrier        string
	TrackingNo     string
	Status         string
	CreatedAt      time.Time
	ShipRecipient  string
	ShipPostalCode string
	ShipPrefecture string
	ShipCity       string
	ShipLine1      string
}

type ShipmentCreate struct {
	OrderID    int64
	Carrier    string
	TrackingNo string
	Status     string
}

type ShipmentUpdate struct {
	ID     int64
	Status string
}

func toShipment(r db.ShippingShipment) Shipment {
	return Shipment{
		ID:             r.ID,
		OrderID:        r.OrderID,
		Carrier:        r.Carrier,
		TrackingNo:     r.TrackingNo,
		Status:         r.Status,
		CreatedAt:      r.CreatedAt.Time,
		ShipRecipient:  r.ShipRecipient,
		ShipPostalCode: r.ShipPostalCode,
		ShipPrefecture: r.ShipPrefecture,
		ShipCity:       r.ShipCity,
		ShipLine1:      r.ShipLine1,
	}
}

func toShipments(rows []db.ShippingShipment) []Shipment {
	out := make([]Shipment, 0, len(rows))
	for _, r := range rows {
		out = append(out, toShipment(r))
	}
	return out
}
