package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

const dbEnv = "DATABASE_URL_OPS"

func seedShipments(t *testing.T, pool *pgxpool.Pool, rows ...db.ShippingShipment) {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE shipping.shipments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO shipping.shipments (order_id, carrier, tracking_no, status) VALUES ($1, $2, $3, $4)`,
			r.OrderID, r.Carrier, r.TrackingNo, r.Status); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func TestRepositoryListShipments(t *testing.T) {
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
	r := NewRepository(pool)
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

func TestRepositoryListShipmentsError(t *testing.T) {
	skip.Short(t)
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListShipments(ctx); err == nil {
		t.Fatal("ListShipments: want error from canceled context")
	}
}

func TestRepositoryGetShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
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

func TestRepositoryCreateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	seedShipments(t, pool)

	got, err := r.CreateShipment(t.Context(), db.CreateShipmentParams{OrderID: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending"})
	if err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if got.ID == 0 || got.TrackingNo != "TRK-10" {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestRepositoryUpdateShipment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	seedShipments(t, pool, db.ShippingShipment{OrderID: 200, Carrier: "佐川急便", TrackingNo: "TRK-10", Status: "pending"})

	t.Run("正常系 既存行を更新して返す (order_id は不変)", func(t *testing.T) {
		got, err := r.UpdateShipment(t.Context(), db.UpdateShipmentParams{ID: 1, Carrier: "ヤマト運輸", TrackingNo: "TRK-99", Status: "delivered"})
		if err != nil {
			t.Fatalf("UpdateShipment: %v", err)
		}
		if got.ID != 1 || got.Carrier != "ヤマト運輸" || got.TrackingNo != "TRK-99" || got.Status != "delivered" || got.OrderID != 200 {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateShipment(t.Context(), db.UpdateShipmentParams{ID: 9999, Carrier: "x", TrackingNo: "x", Status: "x"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
