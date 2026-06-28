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

type refunderStub struct {
	got []int64
	err error
}

func (s *refunderStub) RefundByOrder(_ context.Context, orderID int64) error {
	s.got = append(s.got, orderID)
	return s.err
}

func newTestConsumer(t *testing.T, refunder PaymentRefunder) (*Consumer, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	c := New(rc, refunder)
	c.block = 10 * time.Millisecond
	return c, rc
}

func TestEnsureGroup(t *testing.T) {
	c, _ := newTestConsumer(t, &refunderStub{})
	ctx := t.Context()

	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup (1st): %v", err)
	}
	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup (2nd): %v", err)
	}
}

func TestReadAndProcess(t *testing.T) {
	type args struct {
		values      map[string]any
		refunderErr error
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
			"正常系 order.cancelled で返金し ack する",
			args{map[string]any{"event": "order.cancelled", "orderId": "20"}, nil},
			want{[]int64{20}, 0},
		},
		{
			"準正常系 関心外イベントは返金せず ack する",
			args{map[string]any{"event": "order.created", "orderId": "20"}, nil},
			want{nil, 0},
		},
		{
			"準正常系 不正な orderId は返金せず ack する",
			args{map[string]any{"event": "order.cancelled", "orderId": "abc"}, nil},
			want{nil, 0},
		},
		{
			"異常系 返金が他のエラーなら ack せず pending に残す",
			args{map[string]any{"event": "order.cancelled", "orderId": "20"}, errors.New("db down")},
			want{[]int64{20}, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refunder := &refunderStub{err: tt.args.refunderErr}
			c, rc := newTestConsumer(t, refunder)
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

			if len(refunder.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("refunder called with %v, want %v", refunder.got, tt.want.gotOrderIDs)
			}
			for i, id := range tt.want.gotOrderIDs {
				if refunder.got[i] != id {
					t.Fatalf("refunder called with %v, want %v", refunder.got, tt.want.gotOrderIDs)
				}
			}
			p, err := rc.XPending(ctx, orderevent.Stream, consumerGroup).Result()
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
	c, _ := newTestConsumer(t, &refunderStub{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := c.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
}
