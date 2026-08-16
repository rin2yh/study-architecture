package rdb

import (
	"testing"

	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/orderid"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
)

func TestAvailable(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewInventoryCommand(pool)
	q := NewInventoryQuery(pool)
	ctx := t.Context()

	t.Run("準正常系 入庫の無い商品は 0", func(t *testing.T) {
		if got := mustAvail(t, q, ctx, 999); got != 0 {
			t.Fatalf("available = %d, want 0", got)
		}
	})

	t.Run("正常系 期限切れ予約は利用可能在庫を減らさない", func(t *testing.T) {
		if _, err := cmd.StockIn(ctx, 300, 10); err != nil {
			t.Fatalf("StockIn: %v", err)
		}
		if err := cmd.Reserve(ctx, 1, []ReserveLine{{ProductID: 300, Quantity: 3}}); err != nil {
			t.Fatalf("Reserve active: %v", err)
		}
		if err := cmd.Reserve(ctx, 2, []ReserveLine{{ProductID: 300, Quantity: 5}}); err != nil {
			t.Fatalf("Reserve expired: %v", err)
		}
		expire(t, pool, 2)
		if got := mustAvail(t, q, ctx, 300); got != 7 {
			t.Fatalf("available = %d, want 7 (expired excluded)", got)
		}
	})
}

func TestReservationsByOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewInventoryCommand(pool)
	q := NewInventoryQuery(pool)
	ctx := t.Context()

	if _, err := cmd.StockIn(ctx, 400, 10); err != nil {
		t.Fatalf("StockIn: %v", err)
	}

	statesOf := func(t *testing.T, orderID int64) []string {
		t.Helper()
		rows, err := q.ReservationsByOrder(ctx, orderID)
		if err != nil {
			t.Fatalf("ReservationsByOrder: %v", err)
		}
		states := make([]string, 0, len(rows))
		for _, r := range rows {
			states = append(states, r.State)
		}
		return states
	}

	t.Run("準正常系 予約の無い注文は空", func(t *testing.T) {
		if got := statesOf(t, 998); len(got) != 0 {
			t.Fatalf("states = %v, want empty", got)
		}
	})

	t.Run("正常系 未確定は pending", func(t *testing.T) {
		if err := cmd.Reserve(ctx, 10, []ReserveLine{{ProductID: 400, Quantity: 1}}); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if got := statesOf(t, 10); len(got) != 1 || got[0] != "pending" {
			t.Fatalf("states = %v, want [pending]", got)
		}
	})

	t.Run("正常系 確定は confirmed", func(t *testing.T) {
		if err := cmd.ConfirmReservationsByOrder(ctx, orderid.Must(t, "10")); err != nil {
			t.Fatalf("ConfirmReservationsByOrder: %v", err)
		}
		if got := statesOf(t, 10); len(got) != 1 || got[0] != "confirmed" {
			t.Fatalf("states = %v, want [confirmed]", got)
		}
	})

	t.Run("正常系 確定後の取り消しは cancelled", func(t *testing.T) {
		if err := cmd.CompensateByOrder(ctx, orderid.Must(t, "10")); err != nil {
			t.Fatalf("CompensateByOrder: %v", err)
		}
		if got := statesOf(t, 10); len(got) != 1 || got[0] != "cancelled" {
			t.Fatalf("states = %v, want [cancelled]", got)
		}
	})

	t.Run("正常系 未確定の解放は released", func(t *testing.T) {
		if err := cmd.Reserve(ctx, 11, []ReserveLine{{ProductID: 400, Quantity: 1}}); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if err := cmd.CompensateByOrder(ctx, orderid.Must(t, "11")); err != nil {
			t.Fatalf("CompensateByOrder: %v", err)
		}
		if got := statesOf(t, 11); len(got) != 1 || got[0] != "released" {
			t.Fatalf("states = %v, want [released]", got)
		}
	})

	t.Run("正常系 期限切れは expired", func(t *testing.T) {
		if err := cmd.Reserve(ctx, 12, []ReserveLine{{ProductID: 400, Quantity: 1}}); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		expire(t, pool, 12)
		if err := cmd.ExpireReservations(ctx); err != nil {
			t.Fatalf("ExpireReservations: %v", err)
		}
		if got := statesOf(t, 12); len(got) != 1 || got[0] != "expired" {
			t.Fatalf("states = %v, want [expired]", got)
		}
	})
}
