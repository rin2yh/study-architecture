package rdb

import (
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

func TestCreateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewOrderCommand(pool)
	seedOrders(t, pool)

	got, err := r.CreateOrder(t.Context(), db.CreateOrderParams{MemberID: 20, Status: "pending", TotalCents: 1980})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if got.ID == 0 || got.MemberID != 20 {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestUpdateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewOrderCommand(pool)
	seedOrders(t, pool, db.OrderOrder{MemberID: 10, Status: "pending", TotalCents: 1980})

	t.Run("正常系 status のみ更新し member_id/total_cents は不変", func(t *testing.T) {
		got, err := r.UpdateOrder(t.Context(), db.UpdateOrderParams{ID: 1, Status: "paid"})
		if err != nil {
			t.Fatalf("UpdateOrder: %v", err)
		}
		if got.ID != 1 || got.Status != "paid" || got.MemberID != 10 || got.TotalCents != 1980 {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateOrder(t.Context(), db.UpdateOrderParams{ID: 9999, Status: "paid"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
