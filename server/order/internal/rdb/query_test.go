package rdb

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

const dbEnv = "DATABASE_URL_ORDER"

func seedOrders(t *testing.T, pool *pgxpool.Pool, rows ...db.OrderOrder) {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE "order".order_items, "order".orders RESTART IDENTITY`); err != nil {
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

func TestListOrdersByMember(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewOrderQuery(pool)
	seedOrders(t, pool,
		db.OrderOrder{MemberID: 10, Status: "paid", TotalCents: 5000},
		db.OrderOrder{MemberID: 20, Status: "paid", TotalCents: 3000},
		db.OrderOrder{MemberID: 10, Status: "pending", TotalCents: 1980},
	)

	t.Run("正常系 指定会員の注文だけを id 昇順で返す", func(t *testing.T) {
		got, err := r.ListOrdersByMember(t.Context(), 10)
		if err != nil {
			t.Fatalf("ListOrdersByMember: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		for _, o := range got {
			if o.MemberID != 10 {
				t.Fatalf("memberID = %d, want 10", o.MemberID)
			}
		}
	})

	t.Run("準正常系 注文の無い会員は空スライス (nil でない)", func(t *testing.T) {
		got, err := r.ListOrdersByMember(t.Context(), 999)
		if err != nil {
			t.Fatalf("ListOrdersByMember: %v", err)
		}
		if got == nil {
			t.Fatal("want non-nil slice (emit_empty_slices)")
		}
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})
}

func TestListOrders(t *testing.T) {
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
	r := NewOrderQuery(pool)
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
			assert.DeepEqualSlice(t, tt.seed, got, "ID", "CreatedAt")
		})
	}
}

func TestListOrdersError(t *testing.T) {
	skip.Short(t)
	r := NewOrderQuery(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListOrders(ctx); err == nil {
		t.Fatal("ListOrders: want error from canceled context")
	}
}

func TestGetOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewOrderQuery(pool)
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
	t.Run("準正常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetOrder(t.Context(), 9999); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestGetOrderItems(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewOrderQuery(pool)

	t.Run("正常系 明細を id 昇順で返す", func(t *testing.T) {
		seedOrders(t, pool, db.OrderOrder{MemberID: 10, Status: "confirmed", TotalCents: 2500})
		ctx := t.Context()
		if _, err := pool.Exec(ctx,
			`INSERT INTO "order".order_items (order_id, product_id, product_name, unit_price_cents, quantity)
			 VALUES (1, 100, 'Widget', 500, 2), (1, 200, 'Gadget', 1500, 1)`); err != nil {
			t.Fatalf("seed items: %v", err)
		}

		got, err := r.GetOrderItems(ctx, 1)
		if err != nil {
			t.Fatalf("GetOrderItems: %v", err)
		}
		if len(got) != 2 || got[0].ProductName != "Widget" || got[1].ProductName != "Gadget" {
			t.Fatalf("unexpected items: %+v", got)
		}
	})

	t.Run("準正常系 明細が無ければ空スライス", func(t *testing.T) {
		seedOrders(t, pool, db.OrderOrder{MemberID: 10, Status: "pending", TotalCents: 1980})

		got, err := r.GetOrderItems(t.Context(), 1)
		if err != nil {
			t.Fatalf("GetOrderItems: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("want no items, got %+v", got)
		}
	})
}
