package rdb

import (
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

func TestCreateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewShipmentCommand(pool)
	seedShipments(t, pool)

	got, err := r.CreateShipment(t.Context(), db.CreateShipmentParams{OrderID: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending"})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if got.ID == 0 || got.TrackingNo != "TRK-10" {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestUpdateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewShipmentCommand(pool)
	seedShipments(t, pool, db.ShippingShipment{OrderID: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending"})

	t.Run("正常系 status のみ更新し order_id/carrier/tracking_no は不変", func(t *testing.T) {
		got, err := r.UpdateShipment(t.Context(), db.UpdateShipmentParams{ID: 1, Status: "delivered"})
		if err != nil {
			t.Fatalf("UpdateShipment: %v", err)
		}
		if got.ID != 1 || got.Status != "delivered" || got.OrderID != 200 || got.Carrier != "佐川急便" || got.TrackingNo != "TRK-10" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateShipment(t.Context(), db.UpdateShipmentParams{ID: 9999, Status: "x"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
