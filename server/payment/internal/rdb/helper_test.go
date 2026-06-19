package rdb

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rin2yh/study-architecture/server/payment/internal/db"
)

const dbEnv = "DATABASE_URL_CUSTOMER"

func seedPayments(t *testing.T, pool *pgxpool.Pool, rows ...db.PaymentPayment) {
	t.Helper()
	ctx := t.Context()
	if _, err := pool.Exec(ctx, `TRUNCATE payment.payments RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx,
			`INSERT INTO payment.payments (order_id, amount_cents, method, status) VALUES ($1, $2, $3, $4)`,
			r.OrderID, r.AmountCents, r.Method, r.Status); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}
