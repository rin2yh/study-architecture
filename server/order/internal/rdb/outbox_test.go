package rdb

import (
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/orderevent"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

func TestOutboxStore(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewOrderCommand(pool)
	store := NewOutboxStore(pool)
	seedOrders(t, pool, db.OrderOrder{MemberID: 10, Status: "confirmed", TotalCents: 1980})

	if _, err := cmd.CancelOrder(t.Context(), 1, "tp-xyz"); err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}

	t.Run("正常系 未送信を traceparent 付きで取得する", func(t *testing.T) {
		msgs, err := store.FetchUnpublished(t.Context(), 10)
		if err != nil {
			t.Fatalf("FetchUnpublished: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("len = %d, want 1", len(msgs))
		}
		m := msgs[0]
		if m.Stream != orderevent.Stream {
			t.Fatalf("stream = %q, want %q", m.Stream, orderevent.Stream)
		}
		if m.Values[orderevent.FieldEvent] != orderevent.TypeCancelled {
			t.Fatalf("event = %v, want %q", m.Values[orderevent.FieldEvent], orderevent.TypeCancelled)
		}
		if m.Values[orderevent.FieldOrderID] != int64(1) {
			t.Fatalf("orderId = %v, want 1", m.Values[orderevent.FieldOrderID])
		}
		if m.Values[orderevent.FieldTraceparent] != "tp-xyz" {
			t.Fatalf("traceparent = %v, want tp-xyz", m.Values[orderevent.FieldTraceparent])
		}
	})

	t.Run("正常系 送信済みマーク後は取得されない", func(t *testing.T) {
		if err := store.MarkPublished(t.Context(), 1); err != nil {
			t.Fatalf("MarkPublished: %v", err)
		}
		msgs, err := store.FetchUnpublished(t.Context(), 10)
		if err != nil {
			t.Fatalf("FetchUnpublished: %v", err)
		}
		if len(msgs) != 0 {
			t.Fatalf("len = %d, want 0 after publish", len(msgs))
		}
	})
}
