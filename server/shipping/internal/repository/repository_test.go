package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても実 SQL が
// schema と噛み合うかは検証できない。DATABASE_URL_OPS が指す実 DB (compose の db-ops /
// CI の service) へ接続して結合テストする。skip 条件は testdb 参照。
const dbEnv = "DATABASE_URL_OPS"

func seedShipments(t *testing.T, pool *pgxpool.Pool, rows ...db.ShippingShipment) {
	t.Helper()
	ctx := context.Background()
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
	type args struct {
		seed []db.ShippingShipment
	}
	type want struct {
		trackingNos []string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			args: args{seed: []db.ShippingShipment{
				{OrderID: 1, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "shipped"},
				{OrderID: 2, Carrier: "佐川急便", TrackingNo: "TRK-2", Status: "pending"},
			}},
			want: want{trackingNos: []string{"TRK-1", "TRK-2"}},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			args: args{seed: nil},
			want: want{trackingNos: []string{}},
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedShipments(t, pool, tt.args.seed...)

			got, err := r.ListShipments(context.Background())
			if err != nil {
				t.Fatalf("ListShipments: %v", err)
			}
			if got == nil {
				t.Fatal("ListShipments: want non-nil slice (emit_empty_slices)")
			}
			if len(got) != len(tt.want.trackingNos) {
				t.Fatalf("len = %d, want %d (%+v)", len(got), len(tt.want.trackingNos), got)
			}
			for i, no := range tt.want.trackingNos {
				if got[i].TrackingNo != no {
					t.Fatalf("rows[%d].TrackingNo = %q, want %q", i, got[i].TrackingNo, no)
				}
			}
		})
	}
}

func TestRepositoryListShipmentsError(t *testing.T) {
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ListShipments(ctx); err == nil {
		t.Fatal("ListShipments: want error from canceled context")
	}
}

func TestNewPool(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := NewPool(context.Background()); err == nil {
		t.Fatal("NewPool: want error when DATABASE_URL is empty")
	}

	// ダミー DSN。pgxpool.New は遅延接続なので実際の接続は行われず error にならない。
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	pool, err := NewPool(context.Background())
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	if pool == nil {
		t.Fatal("NewPool: pool is nil")
	}
}

func TestNewRepository(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	pool, err := NewPool(context.Background())
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	if NewRepository(pool) == nil {
		t.Fatal("NewRepository: want non-nil")
	}
}
