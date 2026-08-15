package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
	"github.com/rin2yh/study-architecture/server/internal/test/msgtest"
)

type refunderStub struct {
	got []string
	err error
}

func (s *refunderStub) RefundByOrder(_ context.Context, orderID order.ID) error {
	s.got = append(s.got, orderID.String())
	return s.err
}

func TestRun(t *testing.T) {
	cancelled := map[string]any{orderevent.FieldEvent: orderevent.TypeCancelled, order.FieldID: "20"}
	type args struct {
		values      map[string]any
		refunderErr error
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
		{"正常系 order.cancelled を返金して ack する", args{cancelled, nil}, want{[]string{"20"}, 1}},
		{
			"準正常系 別のイベント種別は処理せず ack する",
			args{map[string]any{orderevent.FieldEvent: "order.created", order.FieldID: "20"}, nil},
			want{nil, 1},
		},
		{
			"準正常系 壊れた payload は ack せずブローカの隔離に委ねる",
			args{map[string]any{orderevent.FieldEvent: orderevent.TypeCancelled, order.FieldID: "abc"}, nil},
			want{nil, 0},
		},
		{"異常系 返金が失敗した分は ack しない", args{cancelled, errors.New("db down")}, want{[]string{"20"}, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			refunder := &refunderStub{err: tt.args.refunderErr}
			sub := msgtest.NewSubscriber(cancel, tt.args.values)

			if err := New(sub, refunder).Run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("Run() = %v, want context.Canceled", err)
			}

			if sub.Topic != orderevent.Topic || sub.Queue != queue {
				t.Fatalf("subscribed to (%q, %q), want (%q, %q)", sub.Topic, sub.Queue, orderevent.Topic, queue)
			}
			if len(refunder.got) != len(tt.want.gotOrderIDs) {
				t.Fatalf("refunder called with %v, want %v", refunder.got, tt.want.gotOrderIDs)
			}
			for i, id := range tt.want.gotOrderIDs {
				if refunder.got[i] != id {
					t.Fatalf("refunder called with %v, want %v", refunder.got, tt.want.gotOrderIDs)
				}
			}
			if len(sub.Acked) != tt.want.acked {
				t.Fatalf("acked = %d, want %d", len(sub.Acked), tt.want.acked)
			}
		})
	}
}
