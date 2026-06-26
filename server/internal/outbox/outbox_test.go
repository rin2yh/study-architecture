package outbox

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

func newTestRelay(t *testing.T, store Store) (*Relay, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	r := NewRelay(rc, store)
	r.interval = 10 * time.Millisecond
	return r, rc
}

func TestRelayDrain(t *testing.T) {
	msg := func(id int64) Message {
		return Message{ID: id, Stream: "payment.events", Values: map[string]any{"event": "payment.settled", "id": id}}
	}
	type args struct{ store *fakeStore }
	type want struct {
		streamLen int64
		published []int64
		err       bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 コミット後クラッシュで未送信に残った行をリレーが送出し published にする",
			args{&fakeStore{pending: []Message{msg(1), msg(2)}}},
			want{2, []int64{1, 2}, false},
		},
		{
			"準正常系 未送信が無ければ何も送出しない",
			args{&fakeStore{}},
			want{0, nil, false},
		},
		{
			"異常系 Fetch 失敗時は送出せず error を返す",
			args{&fakeStore{pending: []Message{msg(1)}, fetchErr: errors.New("db down")}},
			want{0, nil, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, rc := newTestRelay(t, tt.args.store)
			err := r.drain(t.Context())
			if tt.want.err && err == nil {
				t.Fatal("drain(): want error")
			}
			if !tt.want.err && err != nil {
				t.Fatalf("drain() = %v, want nil", err)
			}

			n, qerr := rc.XLen(t.Context(), "payment.events").Result()
			if qerr != nil {
				t.Fatalf("XLen: %v", qerr)
			}
			if n != tt.want.streamLen {
				t.Fatalf("stream length = %d, want %d", n, tt.want.streamLen)
			}
			if got := len(tt.args.store.published); got != len(tt.want.published) {
				t.Fatalf("published count = %d, want %d", got, len(tt.want.published))
			}
		})
	}
}

func TestRelayDrainMarkFailureResendsAtLeastOnce(t *testing.T) {
	store := &fakeStore{pending: []Message{{ID: 1, Stream: "payment.events", Values: map[string]any{"id": int64(1)}}}, markErr: errors.New("mark down")}
	r, rc := newTestRelay(t, store)

	if err := r.drain(t.Context()); err == nil {
		t.Fatal("drain(): want error from MarkPublished")
	}
	store.markErr = nil
	if err := r.drain(t.Context()); err != nil {
		t.Fatalf("drain() retry = %v, want nil", err)
	}

	n, err := rc.XLen(t.Context(), "payment.events").Result()
	if err != nil {
		t.Fatalf("XLen: %v", err)
	}
	if n != 2 {
		t.Fatalf("stream length = %d, want 2 (再送で重複送出される)", n)
	}
}

func TestRelayRunStopsOnContextCancel(t *testing.T) {
	r, _ := newTestRelay(t, &fakeStore{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
}
