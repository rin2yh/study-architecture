package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

// repository 層は sqlc 生成クエリへ委譲するだけの薄い層なので、フェイクで通しても実 SQL が
// schema と噛み合うかは検証できない。List は DATABASE_URL_OPS が指す実 DB へ接続して結合
// テストする。skip 条件は testdb 参照。
const dbEnv = "DATABASE_URL_OPS"

// fakeQuerier は Get/Create のエラー正規化 (dberr) を DB なしで検証するための注入点。
type fakeQuerier struct {
	rows     []db.ShippingShipment
	shipment db.ShippingShipment
	err      error
}

func (f fakeQuerier) ListShipments(context.Context) ([]db.ShippingShipment, error) {
	return f.rows, f.err
}

func (f fakeQuerier) GetShipment(context.Context, int64) (db.ShippingShipment, error) {
	return f.shipment, f.err
}

func (f fakeQuerier) CreateShipment(context.Context, db.CreateShipmentParams) (db.ShippingShipment, error) {
	return f.shipment, f.err
}

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

			got, err := r.ListShipments(context.Background())
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
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ListShipments(ctx); err == nil {
		t.Fatal("ListShipments: want error from canceled context")
	}
}

func TestRepositoryGetShipment(t *testing.T) {
	shipment := db.ShippingShipment{ID: 1, TrackingNo: "TRK-1"}
	other := errors.New("query failed")
	type args struct{ q fakeQuerier }
	type want struct {
		id  int64
		err error // errors.Is で照合。nil は成功
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 行を返す", args{fakeQuerier{shipment: shipment}}, want{1, nil}},
		{"異常系 no rows は ErrNotFound に正規化", args{fakeQuerier{err: pgx.ErrNoRows}}, want{0, dberr.ErrNotFound}},
		{"異常系 その他エラーは透過", args{fakeQuerier{err: other}}, want{0, other}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).GetShipment(context.Background(), 1)
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetShipment: %v", err)
			}
			if got.ID != tt.want.id {
				t.Fatalf("id = %d, want %d", got.ID, tt.want.id)
			}
		})
	}
}

func TestRepositoryCreateShipment(t *testing.T) {
	created := db.ShippingShipment{ID: 10, TrackingNo: "TRK-10"}
	other := errors.New("query failed")
	type args struct{ q fakeQuerier }
	type want struct {
		id  int64
		err error // errors.Is で照合。nil は成功
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 作成行を返す", args{fakeQuerier{shipment: created}}, want{10, nil}},
		{"異常系 その他エラーは透過", args{fakeQuerier{err: other}}, want{0, other}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Repository{q: tt.args.q}).CreateShipment(context.Background(), db.CreateShipmentParams{})
			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err = %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateShipment: %v", err)
			}
			if got.ID != tt.want.id {
				t.Fatalf("id = %d, want %d", got.ID, tt.want.id)
			}
		})
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
