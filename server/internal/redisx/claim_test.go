package redisx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/internal/redisx"
)

const (
	testStream   = "test:events"
	testGroup    = "test-group"
	failedWorker = "worker-1"
	nextWorker   = "worker-2"
)

// claimMinIdle は非公開なので、テストからは miniredis の時計を進めて経過を作る。
var base = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

// newPending は「読んだが ack しないまま落ちた consumer」の PEL を作る。
func newPending(t *testing.T, values ...map[string]any) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	mr.SetTime(base)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })

	ctx := t.Context()
	if err := rc.XGroupCreateMkStream(ctx, testStream, testGroup, "0").Err(); err != nil {
		t.Fatalf("XGroupCreateMkStream: %v", err)
	}
	for _, v := range values {
		if err := rc.XAdd(ctx, &redis.XAddArgs{Stream: testStream, Values: v}).Err(); err != nil {
			t.Fatalf("XAdd: %v", err)
		}
	}
	if len(values) == 0 {
		return mr, rc
	}
	if _, err := rc.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    testGroup,
		Consumer: failedWorker,
		Streams:  []string{testStream, ">"},
		Count:    int64(len(values)),
	}).Result(); err != nil {
		t.Fatalf("XReadGroup: %v", err)
	}
	return mr, rc
}

func pendingCount(t *testing.T, rc *redis.Client) int64 {
	t.Helper()
	p, err := rc.XPending(t.Context(), testStream, testGroup).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	return p.Count
}

func TestClaimPending(t *testing.T) {
	type args struct {
		idleFor    time.Duration
		processErr error
	}
	type want struct {
		processed int
		pending   int64
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 min-idle 経過後の未 ACK は引き取って再処理し ack する",
			args{31 * time.Second, nil},
			want{1, 0},
		},
		{
			"準正常系 min-idle 未経過は処理中とみなし引き取らない",
			args{10 * time.Second, nil},
			want{0, 1},
		},
		{
			"異常系 再処理が失敗した分は ack せず PEL に残す",
			args{31 * time.Second, errors.New("still down")},
			want{1, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr, rc := newPending(t, map[string]any{"event": "payment.settled", "orderId": "20"})
			mr.SetTime(base.Add(tt.args.idleFor))

			var got []string
			err := redisx.ClaimPending(t.Context(), rc, testStream, testGroup, nextWorker,
				func(_ context.Context, id string, values map[string]any) error {
					got = append(got, values["orderId"].(string))
					return tt.args.processErr
				})
			if err != nil {
				t.Fatalf("ClaimPending: %v", err)
			}

			if len(got) != tt.want.processed {
				t.Fatalf("processed %v, want %d messages", got, tt.want.processed)
			}
			if n := pendingCount(t, rc); n != tt.want.pending {
				t.Fatalf("pending = %d, want %d", n, tt.want.pending)
			}
		})
	}
}

// PEL が 1 バッチ (claimCount) を超えても、カーソルを進めて全件引き取れること。
func TestClaimPendingDrainsBeyondOneBatch(t *testing.T) {
	const total = 20
	values := make([]map[string]any, 0, total)
	for i := range total {
		values = append(values, map[string]any{"event": "payment.settled", "orderId": fmt.Sprint(i)})
	}
	mr, rc := newPending(t, values...)
	mr.SetTime(base.Add(31 * time.Second))

	processed := 0
	err := redisx.ClaimPending(t.Context(), rc, testStream, testGroup, nextWorker,
		func(_ context.Context, _ string, _ map[string]any) error {
			processed++
			return nil
		})
	if err != nil {
		t.Fatalf("ClaimPending: %v", err)
	}

	if processed != total {
		t.Fatalf("processed = %d, want %d", processed, total)
	}
	if n := pendingCount(t, rc); n != 0 {
		t.Fatalf("pending = %d, want 0", n)
	}
}

func TestClaimPendingNoPending(t *testing.T) {
	mr, rc := newPending(t)
	mr.SetTime(base.Add(31 * time.Second))

	called := false
	err := redisx.ClaimPending(t.Context(), rc, testStream, testGroup, nextWorker,
		func(_ context.Context, _ string, _ map[string]any) error {
			called = true
			return nil
		})
	if err != nil {
		t.Fatalf("ClaimPending: %v", err)
	}
	if called {
		t.Fatal("process called, want no call")
	}
}

func TestClaimPendingStopsOnCanceledContext(t *testing.T) {
	mr, rc := newPending(t, map[string]any{"event": "payment.settled", "orderId": "20"})
	mr.SetTime(base.Add(31 * time.Second))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := redisx.ClaimPending(ctx, rc, testStream, testGroup, nextWorker,
		func(_ context.Context, _ string, _ map[string]any) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ClaimPending() = %v, want context.Canceled", err)
	}
}
