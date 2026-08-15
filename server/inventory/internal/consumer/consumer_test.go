package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/internal/test/msgtest"
)

type confirmerStub struct {
	got []string
	err error
}

func (s *confirmerStub) ConfirmReservationsByOrder(_ context.Context, orderID order.ID) error {
	s.got = append(s.got, orderID.String())
	return s.err
}

func TestRun(t *testing.T) {
	settled := map[string]any{paymentevent.FieldEvent: paymentevent.TypeSettled, order.FieldID: "20"}
	type args struct {
		values map[string]any
		err    error
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
		{"正常系 payment.settled で予約を確定して ack する", args{settled, nil}, want{[]string{"20"}, 1}},
		{
			"準正常系 別のイベント種別は処理せず ack する",
			args{map[string]any{paymentevent.FieldEvent: "payment.failed", order.FieldID: "20"}, nil},
			want{nil, 1},
		},
		{
			"準正常系 壊れた payload は ack せずブローカの隔離に委ねる",
			args{map[string]any{paymentevent.FieldEvent: paymentevent.TypeSettled, order.FieldID: "abc"}, nil},
			want{nil, 0},
		},
		{"異常系 確定が失敗した分は ack しない", args{settled, errors.New("db down")}, want{[]string{"20"}, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			confirmer := &confirmerStub{err: tt.args.err}
			sub := msgtest.NewSubscriber(cancel, tt.args.values)

			if err := New(sub, confirmer).Run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("Run() = %v, want context.Canceled", err)
			}

			if len(confirmer.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("confirmer called with %v, want %v", confirmer.got, tt.want.gotOrderIDs)
			}
			if len(sub.Acked) != tt.want.acked {
				t.Fatalf("acked = %d, want %d", len(sub.Acked), tt.want.acked)
			}
		})
	}
}

func TestRunSubscribesToPaymentEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sub := msgtest.NewSubscriber(cancel)

	if err := New(sub, &confirmerStub{}).Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}

	if sub.Topic != paymentevent.Topic || sub.Queue != queue {
		t.Fatalf("subscribed to (%q, %q), want (%q, %q)", sub.Topic, sub.Queue, paymentevent.Topic, queue)
	}
}
