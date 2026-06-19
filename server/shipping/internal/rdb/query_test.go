package rdb

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

func TestShipmentQueryListShipments(t *testing.T) {
	skip.Short(t)
	tests := []struct {
		name string
		seed []db.ShippingShipment
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			seed: []db.ShippingShipment{
				{OrderID: 1, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "shipped"},
				{OrderID: 2, Carrier: "佐川急便", TrackingNo: "TRK-2", Status: "pending"},
			},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			seed: nil,
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewShipmentQuery(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedShipments(t, pool, tt.seed...)

			got, err := r.ListShipments(t.Context())
			if err != nil {
				t.Fatalf("ListShipments: %v", err)
			}
			if got == nil {
				t.Fatal("ListShipments: want non-nil slice (emit_empty_slices)")
			}
			if diff := cmp.Diff(tt.seed, got,
				cmpopts.IgnoreFields(db.ShippingShipment{}, "ID", "CreatedAt"),
				cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListShipments mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestShipmentQueryListShipmentsError(t *testing.T) {
	skip.Short(t)
	r := NewShipmentQuery(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListShipments(ctx); err == nil {
		t.Fatal("ListShipments: want error from canceled context")
	}
}

func TestShipmentQueryGetShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewShipmentQuery(pool)
	seedShipments(t, pool, db.ShippingShipment{OrderID: 100, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "shipped"})

	t.Run("正常系 既存 id の行を返す", func(t *testing.T) {
		got, err := r.GetShipment(t.Context(), 1)
		if err != nil {
			t.Fatalf("GetShipment: %v", err)
		}
		if got.TrackingNo != "TRK-1" {
			t.Fatalf("trackingNo = %q, want TRK-1", got.TrackingNo)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetShipment(t.Context(), 9999); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
