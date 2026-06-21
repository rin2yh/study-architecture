package rdb

import (
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/test/assert"
	testdb "github.com/rin2yh/study-architecture/server/internal/test/db"
	"github.com/rin2yh/study-architecture/server/internal/test/skip"
	"github.com/rin2yh/study-architecture/server/order/internal/db"
)

func TestCreateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewOrderCommand(pool)
	seedOrders(t, pool)

	got, err := r.CreateOrder(t.Context(), db.CreateOrderParams{MemberID: 20, Status: "pending", TotalCents: 1980})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if got.ID == 0 || got.MemberID != 20 {
		t.Fatalf("unexpected row: %+v", got)
	}
}

func TestUpdateOrder(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	r := NewOrderCommand(pool)
	seedOrders(t, pool, db.OrderOrder{MemberID: 10, Status: "pending", TotalCents: 1980})

	t.Run("正常系 status のみ更新し member_id/total_cents は不変", func(t *testing.T) {
		got, err := r.UpdateOrder(t.Context(), db.UpdateOrderParams{ID: 1, Status: "paid"})
		if err != nil {
			t.Fatalf("UpdateOrder: %v", err)
		}
		if got.ID != 1 || got.Status != "paid" || got.MemberID != 10 || got.TotalCents != 1980 {
			t.Fatalf("unexpected row: %+v", got)
		}
	})
	t.Run("異常系 未存在は ErrNotFound", func(t *testing.T) {
		if _, err := r.UpdateOrder(t.Context(), db.UpdateOrderParams{ID: 9999, Status: "paid"}); !errors.Is(err, dberr.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestCheckout(t *testing.T) {
	skip.Short(t)
	pool := testdb.Open(t, dbEnv)
	c := NewOrderCommand(pool)
	seedOrders(t, pool)

	lines := []CheckoutLine{
		{ProductID: 100, ProductName: "Widget", UnitPriceCents: 500, Quantity: 2},
		{ProductID: 200, ProductName: "Gadget", UnitPriceCents: 1500, Quantity: 1},
	}
	order, items, err := c.Checkout(t.Context(), 20, "confirmed", 2500, lines)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if order.ID == 0 || order.MemberID != 20 || order.Status != "confirmed" || order.TotalCents != 2500 {
		t.Fatalf("unexpected order: %+v", order)
	}
	if len(items) != 2 || items[0].OrderID != order.ID || items[0].ProductName != "Widget" {
		t.Fatalf("unexpected items: %+v", items)
	}

	got, err := NewOrderQuery(pool).GetOrderItems(t.Context(), order.ID)
	if err != nil {
		t.Fatalf("GetOrderItems: %v", err)
	}
	assert.DeepEqualSlice(t, items, got)
}
