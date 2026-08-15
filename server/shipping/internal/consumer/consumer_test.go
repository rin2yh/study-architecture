package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/internal/test/msgtest"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
)

var fullDest = gateway.Destination{Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}

type creatorStub struct {
	got     []string
	gotDest []gateway.Destination
	err     error
}

func (s *creatorStub) CreateShipmentForOrder(_ context.Context, orderID order.ID, dest gateway.Destination) (db.ShippingShipment, error) {
	s.got = append(s.got, orderID.String())
	s.gotDest = append(s.gotDest, dest)
	return db.ShippingShipment{}, s.err
}

type orderStub struct {
	dest gateway.Destination
	err  error
}

func (s *orderStub) FetchDestination(_ context.Context, _ order.ID) (gateway.Destination, error) {
	return s.dest, s.err
}

func TestRun(t *testing.T) {
	settled := map[string]any{paymentevent.FieldEvent: paymentevent.TypeSettled, order.FieldID: "20"}
	type args struct {
		values    map[string]any
		createErr error
		fetchErr  error
	}
	type want struct {
		gotOrderIDs []string
		acked       int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 payment.settled で配送枠を作り ack する", args{settled, nil, nil}, want{[]string{"20"}, 1}},
		{
			"準正常系 別のイベント種別は処理せず ack する",
			args{map[string]any{paymentevent.FieldEvent: "payment.failed", order.FieldID: "20"}, nil, nil},
			want{nil, 1},
		},
		{
			"準正常系 壊れた payload は ack せずブローカの隔離に委ねる",
			args{map[string]any{paymentevent.FieldEvent: paymentevent.TypeSettled, order.FieldID: "abc"}, nil, nil},
			want{nil, 0},
		},
		{
			"準正常系 再配送による二重手配は冪等に ack する (ADR-[[202606261214]])",
			args{settled, dberr.ErrConflict, nil},
			want{[]string{"20"}, 1},
		},
		{"異常系 配送先の取得が失敗した分は ack しない", args{settled, nil, errors.New("upstream down")}, want{nil, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			creator := &creatorStub{err: tt.args.createErr}
			sub := msgtest.NewSubscriber(cancel, tt.args.values)

			c := New(sub, creator, &orderStub{dest: fullDest, err: tt.args.fetchErr})
			if err := c.Run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("Run() = %v, want context.Canceled", err)
			}

			if sub.Topic != paymentevent.Topic || sub.Queue != queue {
				t.Fatalf("subscribed to (%q, %q), want (%q, %q)", sub.Topic, sub.Queue, paymentevent.Topic, queue)
			}
			if len(creator.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("creator called with %v, want %v", creator.got, tt.want.gotOrderIDs)
			}
			if len(creator.gotDest) > 0 && creator.gotDest[0] != fullDest {
				t.Fatalf("dest = %+v, want %+v", creator.gotDest[0], fullDest)
			}
			if len(sub.Acked) != tt.want.acked {
				t.Fatalf("acked = %d, want %d", len(sub.Acked), tt.want.acked)
			}
		})
	}
}
