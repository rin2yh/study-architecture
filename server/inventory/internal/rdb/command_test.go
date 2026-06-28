package rdb

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
)

const dbEnv = "DATABASE_URL_INVENTORY"

func mustAvail(t *testing.T, q *InventoryQuery, ctx context.Context, productID int64) int64 {
	t.Helper()
	got, err := q.Available(ctx, productID)
	if err != nil {
		t.Fatalf("Available: %v", err)
	}
	return got
}

// expires_at 列を持たず created_at + TTL で期限を導出するため (時間経過の代わり)。
func expire(t *testing.T, pool *pgxpool.Pool, orderID int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE inventory.reservations SET created_at = now() - interval '1 hour' WHERE order_id = $1`, orderID); err != nil {
		t.Fatalf("backdate reservation %d: %v", orderID, err)
	}
}

func TestReserve(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewInventoryCommand(pool)
	q := NewInventoryQuery(pool)
	ctx := t.Context()

	if _, err := cmd.StockIn(ctx, 100, 10); err != nil {
		t.Fatalf("StockIn: %v", err)
	}

	t.Run("正常系 在庫内の予約は成功し available が減る", func(t *testing.T) {
		if err := cmd.Reserve(ctx, 1, []ReserveLine{{ProductID: 100, Quantity: 3}}); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if got := mustAvail(t, q, ctx, 100); got != 7 {
			t.Fatalf("available = %d, want 7", got)
		}
	})

	t.Run("準正常系 在庫超過の予約は ErrInsufficientStock で台帳を変えない", func(t *testing.T) {
		if err := cmd.Reserve(ctx, 2, []ReserveLine{{ProductID: 100, Quantity: 8}}); !errors.Is(err, ErrInsufficientStock) {
			t.Fatalf("Reserve over stock err = %v, want ErrInsufficientStock", err)
		}
		if got := mustAvail(t, q, ctx, 100); got != 7 {
			t.Fatalf("available after rejected = %d, want 7", got)
		}
	})
}

func TestReserveMultiLineRollback(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewInventoryCommand(pool)
	q := NewInventoryQuery(pool)
	ctx := t.Context()

	if _, err := cmd.StockIn(ctx, 100, 5); err != nil {
		t.Fatalf("StockIn 100: %v", err)
	}
	if _, err := cmd.StockIn(ctx, 200, 5); err != nil {
		t.Fatalf("StockIn 200: %v", err)
	}

	// 2 明細のうち 1 つでも不足なら tx ごと巻き戻り、先行明細の予約も残らない。
	err := cmd.Reserve(ctx, 1, []ReserveLine{{ProductID: 100, Quantity: 3}, {ProductID: 200, Quantity: 9}})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Reserve err = %v, want ErrInsufficientStock", err)
	}
	if got := mustAvail(t, q, ctx, 100); got != 5 {
		t.Fatalf("product 100 available = %d, want 5 (no partial reservation)", got)
	}
}

func TestReserveConfirmRelease(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewInventoryCommand(pool)
	q := NewInventoryQuery(pool)
	ctx := t.Context()

	if _, err := cmd.StockIn(ctx, 100, 10); err != nil {
		t.Fatalf("StockIn: %v", err)
	}

	t.Run("正常系 解放で利用可能在庫が戻る", func(t *testing.T) {
		if err := cmd.Reserve(ctx, 1, []ReserveLine{{ProductID: 100, Quantity: 4}}); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if err := cmd.ReleaseReservationsByOrder(ctx, 1); err != nil {
			t.Fatalf("Release: %v", err)
		}
		if got := mustAvail(t, q, ctx, 100); got != 10 {
			t.Fatalf("available after release = %d, want 10", got)
		}
	})

	t.Run("正常系 確定後は利用可能在庫を戻さず解放も効かない", func(t *testing.T) {
		if err := cmd.Reserve(ctx, 2, []ReserveLine{{ProductID: 100, Quantity: 4}}); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if err := cmd.ConfirmReservationsByOrder(ctx, 2); err != nil {
			t.Fatalf("Confirm: %v", err)
		}
		// 確定の二重適用は冪等 (ON CONFLICT DO NOTHING)。
		if err := cmd.ConfirmReservationsByOrder(ctx, 2); err != nil {
			t.Fatalf("Confirm again: %v", err)
		}
		if err := cmd.ReleaseReservationsByOrder(ctx, 2); err != nil {
			t.Fatalf("Release after confirm: %v", err)
		}
		if got := mustAvail(t, q, ctx, 100); got != 6 {
			t.Fatalf("available after confirm = %d, want 6", got)
		}
	})
}

func TestExpireReservations(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewInventoryCommand(pool)
	q := NewInventoryQuery(pool)
	ctx := t.Context()

	if _, err := cmd.StockIn(ctx, 100, 10); err != nil {
		t.Fatalf("StockIn: %v", err)
	}
	if err := cmd.Reserve(ctx, 1, []ReserveLine{{ProductID: 100, Quantity: 4}}); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	expire(t, pool, 1)

	if err := cmd.ExpireReservations(ctx); err != nil {
		t.Fatalf("ExpireReservations: %v", err)
	}
	// 主キー衝突で二重に期限切れ記録しない。
	if err := cmd.ExpireReservations(ctx); err != nil {
		t.Fatalf("ExpireReservations again: %v", err)
	}
	if got := mustAvail(t, q, ctx, 100); got != 10 {
		t.Fatalf("available = %d, want 10", got)
	}

	// 終端状態は相互排他 (期限切れ済みは確定できない)。
	if err := cmd.ConfirmReservationsByOrder(ctx, 1); err != nil {
		t.Fatalf("Confirm after expire: %v", err)
	}
	if got := mustAvail(t, q, ctx, 100); got != 10 {
		t.Fatalf("available after confirm-of-expired = %d, want 10 (not confirmed)", got)
	}
}

// 並行 checkout で利用可能在庫がマイナスにならないことを DB で保証する (ADR-[[202606262000]] / ADR-[[202606180902]])。
func TestReserveConcurrentNoOversell(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewInventoryCommand(pool)
	q := NewInventoryQuery(pool)
	ctx := t.Context()

	const stock = 5
	const attempts = 20
	if _, err := cmd.StockIn(ctx, 100, stock); err != nil {
		t.Fatalf("StockIn: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var success, insufficient int
	for i := range attempts {
		wg.Add(1)
		go func(orderID int64) {
			defer wg.Done()
			err := cmd.Reserve(ctx, orderID, []ReserveLine{{ProductID: 100, Quantity: 1}})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				success++
			case errors.Is(err, ErrInsufficientStock):
				insufficient++
			default:
				t.Errorf("Reserve order %d: unexpected error %v", orderID, err)
			}
		}(int64(i + 1))
	}
	wg.Wait()

	if success != stock {
		t.Fatalf("successful reservations = %d, want %d", success, stock)
	}
	if insufficient != attempts-stock {
		t.Fatalf("insufficient reservations = %d, want %d", insufficient, attempts-stock)
	}
	if got := mustAvail(t, q, ctx, 100); got != 0 {
		t.Fatalf("final available = %d, want 0 (never negative)", got)
	}
}
