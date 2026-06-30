package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/internal/orderevent"
)

type compensatorStub struct {
	got []int64
	err error
}

func (s *compensatorStub) CompensateByOrder(_ context.Context, orderID int64) error {
	s.got = append(s.got, orderID)
	return s.err
}

func newTestCancelConsumer(t *testing.T, compensator ReservationCompensator) (*CancelConsumer, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	c := NewCancel(rc, compensator)
	c.block = 10 * time.Millisecond
	return c, rc
}

func TestCancelEnsureGroup(t *testing.T) {
	c, _ := newTestCancelConsumer(t, &compensatorStub{})
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
		values        map[string]any
		compensateErr error
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
			"正常系 order.cancelled で在庫を戻し ack する",
			args{map[string]any{"event": "order.cancelled", "orderId": "20"}, nil},
			want{[]int64{20}, 0},
		},
		{
			"準正常系 関心外イベントは戻さず ack する",
			args{map[string]any{"event": "order.created", "orderId": "20"}, nil},
			want{nil, 0},
		},
		{
			"準正常系 不正な orderId は戻さず ack する",
			args{map[string]any{"event": "order.cancelled", "orderId": "abc"}, nil},
			want{nil, 0},
		},
		{
			"異常系 戻しが他のエラーなら ack せず pending に残す",
			args{map[string]any{"event": "order.cancelled", "orderId": "20"}, errors.New("db down")},
			want{[]int64{20}, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compensator := &compensatorStub{err: tt.args.compensateErr}
			c, rc := newTestCancelConsumer(t, compensator)
			ctx := t.Context()
			if err := c.ensureGroup(ctx); err != nil {
				t.Fatalf("ensureGroup: %v", err)
			}
			if err := rc.XAdd(ctx, &redis.XAddArgs{Stream: orderevent.Stream, Values: tt.args.values}).Err(); err != nil {
				t.Fatalf("XAdd: %v", err)
			}

			if err := c.readAndProcess(ctx); err != nil {
				t.Fatalf("readAndProcess: %v", err)
			}

			if len(compensator.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("compensator called with %v, want %v", compensator.got, tt.want.gotOrderIDs)
			}
			p, err := rc.XPending(ctx, orderevent.Stream, cancelConsumerGroup).Result()
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
	c, _ := newTestCancelConsumer(t, &compensatorStub{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := c.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
}
