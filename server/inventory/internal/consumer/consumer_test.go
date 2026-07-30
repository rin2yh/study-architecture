package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/internal/redisx"
)

type confirmerStub struct {
	got []int64
	err error
}

func (s *confirmerStub) ConfirmReservationsByOrder(_ context.Context, orderID int64) error {
	s.got = append(s.got, orderID)
	return s.err
}

func newTestConsumer(t *testing.T, confirmer ReservationConfirmer) (*Consumer, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	c := New(rc, confirmer)
	c.block = 10 * time.Millisecond
	return c, mr
}

func TestEnsureGroup(t *testing.T) {
	c, _ := newTestConsumer(t, &confirmerStub{})
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
		values       map[string]any
		confirmerErr error
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
			"正常系 payment.settled で確定し ack する",
			args{map[string]any{"event": "payment.settled", "orderId": "20"}, nil},
			want{[]int64{20}, 0},
		},
		{
			"準正常系 関心外イベントは確定せず ack する",
			args{map[string]any{"event": "payment.refunded", "orderId": "20"}, nil},
			want{nil, 0},
		},
		{
			"準正常系 不正な orderId は確定せず ack する",
			args{map[string]any{"event": "payment.settled", "orderId": "abc"}, nil},
			want{nil, 0},
		},
		{
			"異常系 確定が他のエラーなら ack せず pending に残す",
			args{map[string]any{"event": "payment.settled", "orderId": "20"}, errors.New("db down")},
			want{[]int64{20}, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confirmer := &confirmerStub{err: tt.args.confirmerErr}
			c, _ := newTestConsumer(t, confirmer)
			ctx := t.Context()
			if err := c.ensureGroup(ctx); err != nil {
				t.Fatalf("ensureGroup: %v", err)
			}
			if err := c.rdb.XAdd(ctx, &redis.XAddArgs{Stream: paymentevent.Stream, Values: tt.args.values}).Err(); err != nil {
				t.Fatalf("XAdd: %v", err)
			}

			if err := c.readAndProcess(ctx); err != nil {
				t.Fatalf("readAndProcess: %v", err)
			}

			if len(confirmer.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("confirmer called with %v, want %v", confirmer.got, tt.want.gotOrderIDs)
			}
			for i, id := range tt.want.gotOrderIDs {
				if confirmer.got[i] != id {
					t.Fatalf("confirmer called with %v, want %v", confirmer.got, tt.want.gotOrderIDs)
				}
			}
			p, err := c.rdb.XPending(ctx, paymentevent.Stream, consumerGroup).Result()
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
	c, _ := newTestConsumer(t, &confirmerStub{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := c.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
}

// PEL の引き取りは redisx の min-idle 経過が条件なので、miniredis の時計を進めて再現する。
var claimBase = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

// 処理失敗で PEL に残った分が、min-idle 経過後の周回で引き取られ再処理されること。
func TestReadAndProcessClaimsStalePending(t *testing.T) {
	confirmer := &confirmerStub{err: errors.New("db down")}
	c, mr := newTestConsumer(t, confirmer)
	mr.SetTime(claimBase)
	ctx := t.Context()
	if err := c.ensureGroup(ctx); err != nil {
		t.Fatalf("ensureGroup: %v", err)
	}
	if err := c.rdb.XAdd(ctx, &redis.XAddArgs{Stream: paymentevent.Stream, Values: map[string]any{"event": "payment.settled", "orderId": "20"}}).Err(); err != nil {
		t.Fatalf("XAdd: %v", err)
	}
	if err := c.readAndProcess(ctx); err != nil {
		t.Fatalf("readAndProcess (1 周目): %v", err)
	}
	confirmer.err = nil
	mr.SetTime(claimBase.Add(redisx.ClaimMinIdle + time.Second))
	if err := c.readAndProcess(ctx); err != nil {
		t.Fatalf("readAndProcess (2 周目): %v", err)
	}

	if len(confirmer.got) != 2 {
		t.Fatalf("confirmer calls = %d, want 2 (1 周目の失敗 + 引き取り後の再処理)", len(confirmer.got))
	}
	p, err := c.rdb.XPending(ctx, paymentevent.Stream, consumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	if p.Count != 0 {
		t.Fatalf("pending (2 周目) = %d, want 0", p.Count)
	}
}
