package rdb

import (
	"testing"

	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

func TestOutboxStoreFetchUnpublished(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewPaymentCommand(pool)
	store := NewOutboxStore(pool)

	t.Run("正常系 確定マーク済みの行を traceparent 付きで返す", func(t *testing.T) {
		seedPayments(t, pool, db.PaymentPayment{OrderID: 20, AmountCents: 2980, Method: "card", Status: "pending"})
		if _, err := cmd.UpdatePayment(t.Context(), db.UpdatePaymentParams{ID: 1, Status: "paid", MarkSettled: true, Traceparent: "tp-1"}); err != nil {
			t.Fatalf("UpdatePayment: %v", err)
		}

		msgs, err := store.FetchUnpublished(t.Context(), 64)
		if err != nil {
			t.Fatalf("FetchUnpublished: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("len(msgs) = %d, want 1", len(msgs))
		}
		if msgs[0].ID != 1 {
			t.Fatalf("msg ID = %d, want 1", msgs[0].ID)
		}
		if got, _ := msgs[0].Values["orderId"].(int64); got != 20 {
			t.Fatalf("values[orderId] = %v, want 20", msgs[0].Values["orderId"])
		}
		if got, _ := msgs[0].Values["traceparent"].(string); got != "tp-1" {
			t.Fatalf("values[traceparent] = %q, want tp-1", got)
		}
	})

	t.Run("準正常系 未確定 (pending でない) 行は返さない", func(t *testing.T) {
		seedPayments(t, pool, db.PaymentPayment{OrderID: 21, AmountCents: 500, Method: "card", Status: "pending"})

		msgs, err := store.FetchUnpublished(t.Context(), 64)
		if err != nil {
			t.Fatalf("FetchUnpublished: %v", err)
		}
		if len(msgs) != 0 {
			t.Fatalf("len(msgs) = %d, want 0", len(msgs))
		}
	})
}

func TestOutboxStoreMarkPublished(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	cmd := NewPaymentCommand(pool)
	store := NewOutboxStore(pool)

	seedPayments(t, pool, db.PaymentPayment{OrderID: 20, AmountCents: 2980, Method: "card", Status: "pending"})
	if _, err := cmd.UpdatePayment(t.Context(), db.UpdatePaymentParams{ID: 1, Status: "paid", MarkSettled: true, Traceparent: "tp-1"}); err != nil {
		t.Fatalf("UpdatePayment: %v", err)
	}

	if err := store.MarkPublished(t.Context(), 1); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	msgs, err := store.FetchUnpublished(t.Context(), 64)
	if err != nil {
		t.Fatalf("FetchUnpublished: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("len(msgs) = %d, want 0 (送信済みは pending から外れる)", len(msgs))
	}
}
