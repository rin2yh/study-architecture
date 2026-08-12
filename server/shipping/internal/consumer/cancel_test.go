package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
	"github.com/rin2yh/study-architecture/server/internal/redisx"
)

type cancellerStub struct {
	got []int64
	err error
}

func (s *cancellerStub) CancelShipmentForOrder(_ context.Context, orderID order.ID) error {
	s.got = append(s.got, orderID.Int64())
	return s.err
}

func newTestCancelConsumer(t *testing.T, canceller ShipmentCanceller) (*CancelConsumer, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	c := NewCancel(rc, canceller)
	c.block = 10 * time.Millisecond
	return c, mr
}

func TestCancelEnsureGroup(t *testing.T) {
	c, _ := newTestCancelConsumer(t, &cancellerStub{})
	ctx := t.Context()

	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup (1st): %v", err)
	}
	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup (2nd): %v", err)
	}
}

func TestCancelReadAndProcess(t *testing.T) {
	type args struct {
		values    map[string]any
		cancelErr error
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
			"正常系 order.cancelled で配送中止し ack する",
			args{map[string]any{"event": "order.cancelled", "orderId": "20"}, nil},
			want{[]int64{20}, 0},
		},
		{
			"準正常系 関心外イベントは中止せず ack する",
			args{map[string]any{"event": "order.created", "orderId": "20"}, nil},
			want{nil, 0},
		},
		{
			"準正常系 不正な orderId は中止せず ack せず pending に残す",
			args{map[string]any{"event": "order.cancelled", "orderId": "abc"}, nil},
			want{nil, 1},
		},
		{
			"異常系 中止が他のエラーなら ack せず pending に残す",
			args{map[string]any{"event": "order.cancelled", "orderId": "20"}, errors.New("db down")},
			want{[]int64{20}, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canceller := &cancellerStub{err: tt.args.cancelErr}
			c, _ := newTestCancelConsumer(t, canceller)
			ctx := t.Context()
			if err := c.ensureGroup(ctx); err != nil {
				t.Fatalf("ensureGroup: %v", err)
			}
			if err := c.rdb.XAdd(ctx, &redis.XAddArgs{Stream: orderevent.Stream, Values: tt.args.values}).Err(); err != nil {
				t.Fatalf("XAdd: %v", err)
			}

			if err := c.readAndProcess(ctx); err != nil {
				t.Fatalf("readAndProcess: %v", err)
			}

			if len(canceller.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("canceller called with %v, want %v", canceller.got, tt.want.gotOrderIDs)
			}
			p, err := c.rdb.XPending(ctx, orderevent.Stream, cancelConsumerGroup).Result()
			if err != nil {
				t.Fatalf("XPending: %v", err)
			}
			if p.Count != tt.want.pending {
				t.Fatalf("pending = %d, want %d", p.Count, tt.want.pending)
			}
		})
	}
}

func TestCancelRunStopsOnCanceledContext(t *testing.T) {
	c, _ := newTestCancelConsumer(t, &cancellerStub{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := c.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
}

// 処理失敗で PEL に残った分が、min-idle 経過後の周回で引き取られ再処理されること。
func TestCancelReadAndProcessClaimsStalePending(t *testing.T) {
	canceller := &cancellerStub{err: errors.New("db down")}
	c, mr := newTestCancelConsumer(t, canceller)
	mr.SetTime(claimBase)
	ctx := t.Context()
	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup: %v", err)
	}
	if err := c.rdb.XAdd(ctx, &redis.XAddArgs{Stream: orderevent.Stream, Values: map[string]any{"event": "order.cancelled", "orderId": "20"}}).Err(); err != nil {
		t.Fatalf("XAdd: %v", err)
	}
	if err := c.readAndProcess(ctx); err != nil {
		t.Fatalf("readAndProcess (1 周目): %v", err)
	}
	canceller.err = nil
	mr.SetTime(claimBase.Add(redisx.ClaimMinIdle + time.Second))
	if err := c.readAndProcess(ctx); err != nil {
		t.Fatalf("readAndProcess (2 周目): %v", err)
	}

	if len(canceller.got) != 2 {
		t.Fatalf("canceller calls = %d, want 2 (1 周目の失敗 + 引き取り後の再処理)", len(canceller.got))
	}
	p, err := c.rdb.XPending(ctx, orderevent.Stream, cancelConsumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	if p.Count != 0 {
		t.Fatalf("pending (2 周目) = %d, want 0", p.Count)
	}
}
