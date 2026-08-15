package outbox

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

type fakeStore struct {
	pending   []Message
	published []int64
	fetchErr  error
	markErr   error
}

func (s *fakeStore) FetchUnpublished(_ context.Context, limit int) ([]Message, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	out := []Message{}
	for _, m := range s.pending {
		if slices.Contains(s.published, m.ID) {
			continue
		}
		out = append(out, m)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MarkPublished(_ context.Context, id int64) error {
	if s.markErr != nil {
		return s.markErr
	}
	s.published = append(s.published, id)
	return nil
}

// fakePublisher は送出先の代わりに publish されたトピックを記録する。
type fakePublisher struct {
	topics []string
}

func (p *fakePublisher) Publish(_ context.Context, topic string, _ map[string]any) error {
	p.topics = append(p.topics, topic)
	return nil
}

func newTestRelay(t *testing.T, store Store) (*Relay, *fakePublisher) {
	t.Helper()
	pub := &fakePublisher{}
	r := NewRelay(pub, store)
	r.interval = 10 * time.Millisecond
	return r, pub
}

func TestRelayDrain(t *testing.T) {
	msg := func(id int64) Message {
		return Message{ID: id, Topic: "payment-events", Values: map[string]any{"event": "payment.settled", "id": id}}
	}
	type args struct{ store *fakeStore }
	type want struct {
		published  []int64
		publishing int
		err        bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 コミット後クラッシュで未送信に残った行をリレーが送出し published にする",
			args{&fakeStore{pending: []Message{msg(1), msg(2)}}},
			want{[]int64{1, 2}, 2, false},
		},
		{
			"準正常系 未送信が無ければ何も送出しない",
			args{&fakeStore{}},
			want{nil, 0, false},
		},
		{
			"異常系 Fetch 失敗時は送出せず error を返す",
			args{&fakeStore{pending: []Message{msg(1)}, fetchErr: errors.New("db down")}},
			want{nil, 0, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, pub := newTestRelay(t, tt.args.store)
			err := r.drain(t.Context())
			if tt.want.err && err == nil {
				t.Fatal("drain(): want error")
			}
			if !tt.want.err && err != nil {
				t.Fatalf("drain() = %v, want nil", err)
			}

			if len(pub.topics) != tt.want.publishing {
				t.Fatalf("published %v, want %d messages", pub.topics, tt.want.publishing)
			}
			if got := len(tt.args.store.published); got != len(tt.want.published) {
				t.Fatalf("published count = %d, want %d", got, len(tt.want.published))
			}
		})
	}

	t.Run("準正常系 MarkPublished 失敗後の再 drain で再送する (at-least-once)", func(t *testing.T) {
		store := &fakeStore{pending: []Message{{ID: 1, Topic: "payment-events", Values: map[string]any{"id": int64(1)}}}, markErr: errors.New("mark down")}
		r, pub := newTestRelay(t, store)

		if err := r.drain(t.Context()); err == nil {
			t.Fatal("drain(): want error from MarkPublished")
		}
		store.markErr = nil
		if err := r.drain(t.Context()); err != nil {
			t.Fatalf("drain() retry = %v, want nil", err)
		}

		if len(pub.topics) != 2 {
			t.Fatalf("published %v, want 2 (再送で重複送出される)", pub.topics)
		}
	})
}

func TestDecodePayload(t *testing.T) {
	type args struct {
		raw         string
		traceparent string
	}
	type want struct {
		values map[string]any
		err    bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 文字列と整数を復元する",
			args{`{"event":"payment.settled","orderId":20,"amountCents":2980}`, ""},
			want{map[string]any{"event": "payment.settled", "orderId": int64(20), "amountCents": int64(2980)}, false},
		},
		{
			"正常系 int64 上限近傍の整数を桁落ちなく復元する",
			args{`{"id":9007199254740993}`, ""},
			want{map[string]any{"id": int64(9007199254740993)}, false},
		},
		{
			"正常系 traceparent を Values に載せ直す",
			args{`{"event":"order.cancelled","orderId":1}`, "tp-1"},
			want{map[string]any{"event": "order.cancelled", "orderId": int64(1), "traceparent": "tp-1"}, false},
		},
		{
			"準正常系 整数でない数値は文字列で復元する",
			args{`{"ratio":1.5}`, ""},
			want{map[string]any{"ratio": "1.5"}, false},
		},
		{
			"異常系 壊れた JSON は error",
			args{`{"event":`, ""},
			want{nil, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodePayload([]byte(tt.args.raw), tt.args.traceparent)
			if tt.want.err {
				if err == nil {
					t.Fatal("DecodePayload(): want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodePayload() = %v, want nil", err)
			}
			if len(got) != len(tt.want.values) {
				t.Fatalf("len = %d, want %d (%#v)", len(got), len(tt.want.values), got)
			}
			for k, w := range tt.want.values {
				if got[k] != w {
					t.Fatalf("values[%q] = %#v (%T), want %#v (%T)", k, got[k], got[k], w, w)
				}
			}
		})
	}
}

type fakeEvent struct {
	typ    string
	aggID  string
	values map[string]any
}

func (e fakeEvent) EventType() string      { return e.typ }
func (e fakeEvent) AggregateID() string    { return e.aggID }
func (e fakeEvent) Values() map[string]any { return e.values }

type fakeInserter struct {
	rows []Row
	err  error
}

func (f *fakeInserter) InsertOutbox(_ context.Context, row Row) error {
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, row)
	return nil
}

func TestDispatch(t *testing.T) {
	ev := func(typ string, id string) fakeEvent {
		return fakeEvent{typ: typ, aggID: id, values: map[string]any{"event": typ, "id": id}}
	}
	type args struct {
		events      []Event
		traceparent string
		insErr      error
	}
	type want struct {
		rows int
		err  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{"正常系 1 件を payload 化して積む", args{[]Event{ev("payment.settled", "7")}, "tp-1", nil}, want{1, false}},
		{"正常系 複数件を順に積む", args{[]Event{ev("a", "1"), ev("b", "2")}, "", nil}, want{2, false}},
		{"準正常系 0 件なら何もしない", args{nil, "tp", nil}, want{0, false}},
		{"異常系 Inserter 失敗で error を返す", args{[]Event{ev("a", "1")}, "", errors.New("insert down")}, want{0, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ins := &fakeInserter{err: tt.args.insErr}
			err := Dispatch(context.Background(), ins, tt.args.traceparent, tt.args.events...)
			if tt.want.err {
				if err == nil {
					t.Fatal("Dispatch(): want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Dispatch() = %v, want nil", err)
			}
			if len(ins.rows) != tt.want.rows {
				t.Fatalf("rows = %d, want %d", len(ins.rows), tt.want.rows)
			}
			for i, r := range ins.rows {
				e := tt.args.events[i]
				if r.EventType != e.EventType() || r.AggregateID != e.AggregateID() {
					t.Fatalf("row[%d] = {%s,%s}, want {%s,%s}", i, r.EventType, r.AggregateID, e.EventType(), e.AggregateID())
				}
				if r.Traceparent != tt.args.traceparent {
					t.Fatalf("row[%d] traceparent = %q, want %q", i, r.Traceparent, tt.args.traceparent)
				}
				vals, derr := DecodePayload(r.Payload, "")
				if derr != nil {
					t.Fatalf("decode payload: %v", derr)
				}
				if vals["event"] != e.EventType() {
					t.Fatalf("row[%d] payload event = %v, want %v", i, vals["event"], e.EventType())
				}
			}
		})
	}
}

func TestRelayRun(t *testing.T) {
	t.Run("準正常系 ctx キャンセルで context.Canceled を返して停止する", func(t *testing.T) {
		r, _ := newTestRelay(t, &fakeStore{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := r.Run(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() = %v, want context.Canceled", err)
		}
	})
}
