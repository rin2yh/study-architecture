package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

// fakeQuerier は db.Querier を満たし、Repository.q へ差し替えて DB なしで検証する。
type fakeQuerier struct {
	rows []db.ShippingShipment
	err  error
}

func (f fakeQuerier) ListShipments(context.Context) ([]db.ShippingShipment, error) {
	return f.rows, f.err
}

func TestRepositoryListShipments(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r := &Repository{q: fakeQuerier{rows: []db.ShippingShipment{
		{ID: 1, OrderID: 100, Carrier: "ヤマト運輸", TrackingNo: "TRK-1", Status: "shipped", CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}},
	}}}

	got, err := r.ListShipments(context.Background())
	if err != nil {
		t.Fatalf("ListShipments: %v", err)
	}
	if len(got) != 1 || got[0].TrackingNo != "TRK-1" {
		t.Fatalf("unexpected rows: %+v", got)
	}
}

func TestRepositoryListShipmentsError(t *testing.T) {
	want := errors.New("query failed")
	r := &Repository{q: fakeQuerier{err: want}}

	if _, err := r.ListShipments(context.Background()); !errors.Is(err, want) {
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
