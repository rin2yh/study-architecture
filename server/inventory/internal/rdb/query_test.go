package rdb

import (
	"testing"

	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
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
