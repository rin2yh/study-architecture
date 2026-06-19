package rdb

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

func TestPaymentQueryListPayments(t *testing.T) {
	skip.Short(t)
	tests := []struct {
		name string
		seed []db.PaymentPayment
	}{
		{
			name: "正常系 id 昇順 (登録順) に複数件返す",
			seed: []db.PaymentPayment{
				{OrderID: 1, AmountCents: 1980, Method: "card", Status: "paid"},
				{OrderID: 2, AmountCents: 2980, Method: "bank", Status: "pending"},
			},
		},
		{
			name: "準正常系 0 件なら空スライス (nil でない)",
			seed: nil,
		},
	}

	pool := testdb.Open(t, dbEnv)
	r := NewPaymentQuery(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedPayments(t, pool, tt.seed...)

			got, err := r.ListPayments(t.Context())
			if err != nil {
				t.Fatalf("ListPayments: %v", err)
			}
			if got == nil {
				t.Fatal("ListPayments: want non-nil slice (emit_empty_slices)")
			}
			if diff := cmp.Diff(tt.seed, got,
				cmpopts.IgnoreFields(db.PaymentPayment{}, "ID", "CreatedAt"),
				cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ListPayments mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPaymentQueryListPaymentsError(t *testing.T) {
	skip.Short(t)
	r := NewPaymentQuery(testdb.Open(t, dbEnv))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.ListPayments(ctx); err == nil {
		t.Fatal("ListPayments: want error from canceled context")
	}
}

func TestPaymentQueryGetPayment(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewPaymentQuery(pool)
	seedPayments(t, pool, db.PaymentPayment{OrderID: 10, AmountCents: 1980, Method: "card", Status: "paid"})

	t.Run("正常系 既存 id の行を返す", func(t *testing.T) {
		got, err := r.GetPayment(t.Context(), 1)
		if err != nil {
			t.Fatalf("GetPayment: %v", err)
		}
		if got.OrderID != 10 {
			t.Fatalf("orderID = %d, want 10", got.OrderID)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.GetPayment(t.Context(), 9999); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
