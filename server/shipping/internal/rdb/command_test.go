package rdb

import (
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
)

func TestCreateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewShipmentCommand(pool)
	seedShipments(t, pool)

	got, err := r.CreateShipment(t.Context(), ShipmentCreate{OrderID: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending"})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if got.ID == 0 || got.TrackingNo != "TRK-10" {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestCreateShipmentForOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewShipmentCommand(pool)
	seedShipments(t, pool)

	dest := gateway.Destination{Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}
	t.Run("正常系 carrier/tracking 未指定で宛先を持つ preparing 枠を作る", func(t *testing.T) {
		got, err := r.CreateShipmentForOrder(t.Context(), 300, dest)
		if err != nil {
			t.Fatalf("CreateShipmentForOrder: %v", err)
		}
		if got.ID == 0 || got.OrderID != 300 || got.Status != "preparing" || got.Carrier != "" || got.TrackingNo != "" {
			t.Fatalf("unexpected row: %+v", got)
		}
		if got.ShipRecipient != "山田太郎" || got.ShipCity != "渋谷区" || got.ShipLine1 != "神宮前1-2-3" {
			t.Fatalf("destination not persisted: %+v", got)
		}
	})
	t.Run("準正常系 同一 order の再手配は ErrConflict (冪等)", func(t *testing.T) {
		if _, err := r.CreateShipmentForOrder(t.Context(), 300, dest); !errors.Is(err, dberr.ErrConflict) {
			t.Fatalf("err = %v, want ErrConflict", err)
		}
	})
}

func TestCancelShipmentForOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewShipmentCommand(pool)
	seedShipments(t, pool,
		Shipment{OrderID: 200, Status: "preparing"},
		Shipment{OrderID: 300, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "shipped"},
	)

	statusOf := func(t *testing.T, id int64) string {
		t.Helper()
		s, err := r.q.GetShipment(t.Context(), id)
		if err != nil {
			t.Fatalf("GetShipment: %v", err)
		}
		return s.Status
	}

	t.Run("正常系 未発送は中止され再実行でも 1 回に収束", func(t *testing.T) {
		if err := r.CancelShipmentForOrder(t.Context(), 200); err != nil {
			t.Fatalf("CancelShipmentForOrder: %v", err)
		}
		if got := statusOf(t, 1); got != "cancelled" {
			t.Fatalf("status = %q, want cancelled", got)
		}
		if err := r.CancelShipmentForOrder(t.Context(), 200); err != nil {
			t.Fatalf("CancelShipmentForOrder again: %v", err)
		}
		if got := statusOf(t, 1); got != "cancelled" {
			t.Fatalf("status after re-run = %q, want cancelled (idempotent)", got)
		}
	})

	t.Run("準正常系 発送済みは中止しない", func(t *testing.T) {
		if err := r.CancelShipmentForOrder(t.Context(), 300); err != nil {
			t.Fatalf("CancelShipmentForOrder: %v", err)
		}
		if got := statusOf(t, 2); got != "shipped" {
			t.Fatalf("status = %q, want shipped (unchanged)", got)
		}
	})

	t.Run("準正常系 未作成は no-op", func(t *testing.T) {
		if err := r.CancelShipmentForOrder(t.Context(), 999); err != nil {
			t.Fatalf("CancelShipmentForOrder missing: %v", err)
		}
	})
}

func TestUpdateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewShipmentCommand(pool)
	seedShipments(t, pool, Shipment{OrderID: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending"})

	t.Run("正常系 status のみ更新し order_id/carrier/tracking_no は不変", func(t *testing.T) {
		got, err := r.UpdateShipment(t.Context(), ShipmentUpdate{ID: 1, Status: "delivered"})
		if err != nil {
			t.Fatalf("UpdateShipment: %v", err)
		}
		if got.ID != 1 || got.Status != "delivered" || got.OrderID != 200 || got.Carrier != "佐川急便" || got.TrackingNo != "TRK-10" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("準正常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateShipment(t.Context(), ShipmentUpdate{ID: 9999, Status: "x"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
