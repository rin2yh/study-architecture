package rdb

import (
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

func TestCreatePayment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewPaymentCommand(pool)
	seedPayments(t, pool)

	got, err := r.CreatePayment(t.Context(), db.CreatePaymentParams{OrderID: 20, AmountCents: 2980, Method: "card", Status: "paid"})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if got.ID == 0 || got.OrderID != 20 {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestUpdatePayment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewPaymentCommand(pool)
	seedPayments(t, pool, db.PaymentPayment{OrderID: 20, AmountCents: 2980, Method: "card", Status: "pending"})

	t.Run("正常系 status のみ更新し order_id/amount_cents/method は不変", func(t *testing.T) {
		got, err := r.UpdatePayment(t.Context(), db.UpdatePaymentParams{ID: 1, Status: "refunded"})
		if err != nil {
			t.Fatalf("UpdatePayment: %v", err)
		}
		if got.ID != 1 || got.Status != "refunded" || got.OrderID != 20 || got.AmountCents != 2980 || got.Method != "card" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("準正常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdatePayment(t.Context(), db.UpdatePaymentParams{ID: 9999, Status: "paid"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
