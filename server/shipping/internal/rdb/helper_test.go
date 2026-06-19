package rdb

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

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
