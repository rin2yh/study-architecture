package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/internal/redisx"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
)

var fullDest = gateway.Destination{Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}

type creatorStub struct {
	got     []int64
	gotDest []gateway.Destination
	err     error
}

func (s *creatorStub) CreateShipmentForOrder(_ context.Context, orderID int64, dest gateway.Destination) (db.ShippingShipment, error) {
	s.got = append(s.got, orderID)
	s.gotDest = append(s.gotDest, dest)
	return db.ShippingShipment{OrderID: orderID}, s.err
}

type orderStub struct {
	dest gateway.Destination
	err  error
}

func (s *orderStub) FetchDestination(_ context.Context, _ int64) (gateway.Destination, error) {
	return s.dest, s.err
}

func newTestConsumer(t *testing.T, creator ShipmentCreator, order gateway.OrderPort) (*Consumer, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	c := New(rc, creator, order)
	c.block = 10 * time.Millisecond
	return c, mr
}

func TestEnsureGroup(t *testing.T) {
	c, _ := newTestConsumer(t, &creatorStub{}, &orderStub{})
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
		orderDest  gateway.Destination
		orderErr   error
		creatorErr error
	}
	type want struct {
		gotOrderIDs []int64
		gotDest     gateway.Destination
		pending     int64
	}
	settled := map[string]any{"event": "payment.settled", "orderId": "20"}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 payment.settled で order から宛先を引き手配し ack する",
			args{settled, fullDest, nil, nil},
			want{[]int64{20}, fullDest, 0},
		},
		{
			"準正常系 既に手配済み (ErrConflict) でも冪等に ack する",
			args{settled, fullDest, nil, dberr.ErrConflict},
			want{[]int64{20}, fullDest, 0},
		},
		{
			"準正常系 関心外イベントは order を引かず手配せず ack する",
			args{map[string]any{"event": "payment.refunded", "orderId": "20"}, fullDest, nil, nil},
			want{nil, gateway.Destination{}, 0},
		},
		{
			"準正常系 不正な orderId は手配せず ack せず pending に残す",
			args{map[string]any{"event": "payment.settled", "orderId": "abc"}, fullDest, nil, nil},
			want{nil, gateway.Destination{}, 1},
		},
		{
			"異常系 order 取得失敗は手配せず ack せず pending に残す",
			args{settled, gateway.Destination{}, errors.New("order down"), nil},
			want{nil, gateway.Destination{}, 1},
		},
		{
			"異常系 手配が他のエラーなら ack せず pending に残す",
			args{settled, fullDest, nil, errors.New("db down")},
			want{[]int64{20}, fullDest, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &creatorStub{err: tt.args.creatorErr}
			c, _ := newTestConsumer(t, creator, &orderStub{dest: tt.args.orderDest, err: tt.args.orderErr})
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

			if len(creator.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("creator called with %v, want %v", creator.got, tt.want.gotOrderIDs)
			}
			for i, id := range tt.want.gotOrderIDs {
				if creator.got[i] != id {
					t.Fatalf("creator called with %v, want %v", creator.got, tt.want.gotOrderIDs)
				}
			}
			if len(creator.gotDest) > 0 && creator.gotDest[0] != tt.want.gotDest {
				t.Fatalf("creator dest = %#v, want %#v", creator.gotDest[0], tt.want.gotDest)
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
	c, _ := newTestConsumer(t, &creatorStub{}, &orderStub{})
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
	creator := &creatorStub{err: errors.New("db down")}
	c, mr := newTestConsumer(t, creator, &orderStub{dest: fullDest})
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
	creator.err = nil
	mr.SetTime(claimBase.Add(redisx.ClaimMinIdle + time.Second))
	if err := c.readAndProcess(ctx); err != nil {
		t.Fatalf("readAndProcess (2 周目): %v", err)
	}

	if len(creator.got) != 2 {
		t.Fatalf("creator calls = %d, want 2 (1 周目の失敗 + 引き取り後の再処理)", len(creator.got))
	}
	p, err := c.rdb.XPending(ctx, paymentevent.Stream, consumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	if p.Count != 0 {
		t.Fatalf("pending (2 周目) = %d, want 0", p.Count)
	}
}
