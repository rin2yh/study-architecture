package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

type creatorStub struct {
	got []int64
	err error
}

func (s *creatorStub) CreateShipmentForOrder(_ context.Context, orderID int64) (db.ShippingShipment, error) {
	s.got = append(s.got, orderID)
	return db.ShippingShipment{OrderID: orderID}, s.err
}

func newTestConsumer(t *testing.T, creator ShipmentCreator) (*Consumer, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	c := New(rc, creator)
	c.block = 10 * time.Millisecond
	return c, rc
}

func TestNewRedisClient(t *testing.T) {
	type args struct{ url string }
	type want struct{ err bool }
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 有効な REDIS_URL で生成できる", args{"redis://localhost:6379"}, want{false}},
		{"異常系 REDIS_URL 未指定は error", args{""}, want{true}},
		{"異常系 不正な URL は error", args{"::not-a-url"}, want{true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REDIS_URL", tt.args.url)
			_, err := NewRedisClient()
			if tt.want.err && err == nil {
				t.Fatal("NewRedisClient(): want error")
			}
			if !tt.want.err && err != nil {
				t.Fatalf("NewRedisClient() = %v, want nil", err)
			}
		})
	}
}

func TestEnsureGroup(t *testing.T) {
	c, _ := newTestConsumer(t, &creatorStub{})
	ctx := t.Context()

	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup (1st): %v", err)
	}
	// 2 回目は BUSYGROUP になるが正常扱い。
	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup (2nd): %v", err)
	}
}

func TestReadAndProcess(t *testing.T) {
	type args struct {
		values     map[string]any
		creatorErr error
	}
	type want struct {
		gotOrderIDs []int64
		pending     int64
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 payment.settled で手配し ack する",
			args{map[string]any{"event": "payment.settled", "orderId": "20"}, nil},
			want{[]int64{20}, 0},
		},
		{
			"準正常系 既に手配済み (ErrConflict) でも冪等に ack する",
			args{map[string]any{"event": "payment.settled", "orderId": "20"}, dberr.ErrConflict},
			want{[]int64{20}, 0},
		},
		{
			"準正常系 関心外イベントは手配せず ack する",
			args{map[string]any{"event": "payment.refunded", "orderId": "20"}, nil},
			want{nil, 0},
		},
		{
			"準正常系 不正な orderId は手配せず ack する",
			args{map[string]any{"event": "payment.settled", "orderId": "abc"}, nil},
			want{nil, 0},
		},
		{
			"異常系 手配が他のエラーなら ack せず pending に残す",
			args{map[string]any{"event": "payment.settled", "orderId": "20"}, errors.New("db down")},
			want{[]int64{20}, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &creatorStub{err: tt.args.creatorErr}
			c, rc := newTestConsumer(t, creator)
			ctx := t.Context()
			if err := c.ensureGroup(ctx); err != nil {
				t.Fatalf("ensureGroup: %v", err)
			}
			if err := rc.XAdd(ctx, &redis.XAddArgs{Stream: streamPaymentEvents, Values: tt.args.values}).Err(); err != nil {
				t.Fatalf("XAdd: %v", err)
			}

			if err := c.readAndProcess(ctx); err != nil {
				t.Fatalf("readAndProcess: %v", err)
			}

			if len(creator.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("creator called with %v, want %v", creator.got, tt.want.gotOrderIDs)
			}
			for i, id := range tt.want.gotOrderIDs {
				if creator.got[i] != id {
					t.Fatalf("creator called with %v, want %v", creator.got, tt.want.gotOrderIDs)
				}
			}
			p, err := rc.XPending(ctx, streamPaymentEvents, consumerGroup).Result()
			if err != nil {
				t.Fatalf("XPending: %v", err)
			}
			if p.Count != tt.want.pending {
				t.Fatalf("pending = %d, want %d", p.Count, tt.want.pending)
			}
		})
	}
}

func TestRunStopsOnCanceledContext(t *testing.T) {
	c, _ := newTestConsumer(t, &creatorStub{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := c.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
}
