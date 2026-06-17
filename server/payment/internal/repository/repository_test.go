package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-service-base-architecture/server/payment/internal/db"
)

// fakeQuerier は db.Querier を満たし、Repository.q へ差し替えて DB なしで検証する。
type fakeQuerier struct {
	rows []db.PaymentPayment
	err  error
}

func (f fakeQuerier) ListPayments(context.Context) ([]db.PaymentPayment, error) {
	return f.rows, f.err
}

func TestRepositoryListPayments(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r := &Repository{q: fakeQuerier{rows: []db.PaymentPayment{
		{ID: 1, OrderID: 10, AmountCents: 1980, Method: "card", Status: "paid", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}}

	got, err := r.ListPayments(context.Background())
	if err != nil {
		t.Fatalf("ListPayments: %v", err)
	}
	if len(got) != 1 || got[0].Method != "card" {
		t.Fatalf("unexpected rows: %+v", got)
	}
}

func TestRepositoryListPaymentsError(t *testing.T) {
	want := errors.New("query failed")
	r := &Repository{q: fakeQuerier{err: want}}

	if _, err := r.ListPayments(context.Background()); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestNewPool(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := NewPool(context.Background()); err == nil {
		t.Fatal("NewPool: want error when DATABASE_URL is empty")
	}

	// ダミー DSN。pgxpool.New は遅延接続なので実際の接続は行われず error にならない。
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	pool, err := NewPool(context.Background())
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	if pool == nil {
		t.Fatal("NewPool: pool is nil")
	}
}

func TestNewRepository(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	pool, err := NewPool(context.Background())
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	if NewRepository(pool) == nil {
		t.Fatal("NewRepository: want non-nil")
	}
}
