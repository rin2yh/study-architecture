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
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

const dbEnv = "DATABASE_URL_CUSTOMER"

func seedOrders(t *testing.T, pool *pgxpool.Pool, rows ...db.OrderOrder) {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".orders RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO "order".orders (member_id, status, total_cents) VALUES ($1, $2, $3)`,
			r.MemberID, r.Status, r.TotalCents); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func TestRepositoryListOrders(t *testing.T) {
	skip.Short(t)
	tests := []struct {
		name string
		seed []db.OrderOrder
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			seed: []db.OrderOrder{
				{MemberID: 1, Status: "pending", TotalCents: 1980},
				{MemberID: 2, Status: "paid", TotalCents: 2980},
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
			seedOrders(t, pool, tt.seed...)

			got, err := r.ListOrders(t.Context())
			if err != nil {
				t.Fatalf("ListOrders: %v", err)
			}
			if got == nil {
				t.Fatal("ListOrders: want non-nil slice (emit_empty_slices)")
			}
			if diff := cmp.Diff(tt.seed, got,
				cmpopts.IgnoreFields(db.OrderOrder{}, "ID", "CreatedAt"),
				cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListOrders mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRepositoryListOrdersError(t *testing.T) {
	skip.Short(t)
	r := NewRepository(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListOrders(ctx); err == nil {
		t.Fatal("ListOrders: want error from canceled context")
	}
}

func TestRepositoryGetOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	seedOrders(t, pool, db.OrderOrder{MemberID: 10, Status: "paid", TotalCents: 5000})

	t.Run("正常系 既存 id の行を返す", func(t *testing.T) {
		got, err := r.GetOrder(t.Context(), 1)
		if err != nil {
			t.Fatalf("GetOrder: %v", err)
		}
		if got.MemberID != 10 {
			t.Fatalf("memberID = %d, want 10", got.MemberID)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetOrder(t.Context(), 9999); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestRepositoryCreateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	seedOrders(t, pool)

	got, err := r.CreateOrder(t.Context(), db.CreateOrderParams{MemberID: 20, Status: "pending", TotalCents: 1980})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if got.ID == 0 || got.MemberID != 20 {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestRepositoryUpdateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewRepository(pool)
	seedOrders(t, pool, db.OrderOrder{MemberID: 10, Status: "pending", TotalCents: 1980})

	t.Run("正常系 既存行を更新して返す (member_id は不変)", func(t *testing.T) {
		got, err := r.UpdateOrder(t.Context(), db.UpdateOrderParams{ID: 1, Status: "paid", TotalCents: 4980})
		if err != nil {
			t.Fatalf("UpdateOrder: %v", err)
		}
		if got.ID != 1 || got.Status != "paid" || got.TotalCents != 4980 || got.MemberID != 10 {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateOrder(t.Context(), db.UpdateOrderParams{ID: 9999, Status: "paid", TotalCents: 1}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
