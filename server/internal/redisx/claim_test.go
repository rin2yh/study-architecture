package redisx

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const (
	testStream   = "test:events"
	testGroup    = "test-group"
	failedWorker = "worker-1"
	nextWorker   = "worker-2"
)

var (
	// 引き取り条件は min-idle の経過なので、miniredis の時計をこの時刻から進めて再現する。
	base    = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	settled = map[string]any{"event": "payment.settled", "orderId": "20"}
)

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
		values     []map[string]any
		idleFor    time.Duration
		processErr error
	}
	type want struct {
		processed int
		pending   int64
	}
	elapsed := ClaimMinIdle + time.Second
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 min-idle 経過後の未 ACK は引き取って再処理し ack する",
			args{[]map[string]any{settled}, elapsed, nil},
			want{1, 0},
		},
		{
			"準正常系 min-idle 未経過は処理中とみなし引き取らない",
			args{[]map[string]any{settled}, ClaimMinIdle / 3, nil},
			want{0, 1},
		},
		{
			"準正常系 PEL が空なら process を呼ばない",
			args{nil, elapsed, nil},
			want{0, 0},
		},
		{
			"異常系 再処理が失敗した分は ack せず PEL に残す",
			args{[]map[string]any{settled}, elapsed, errors.New("still down")},
			want{1, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr, rc := newPending(t, tt.args.values...)
			mr.SetTime(base.Add(tt.args.idleFor))

			var got []string
			err := ClaimPending(t.Context(), rc, testStream, testGroup, nextWorker,
				func(_ context.Context, _ string, values map[string]any) error {
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

// 引き取りの条件は min-idle の経過なので、1 周ごとに miniredis の時計を進める。
func claimUntilDLQ(t *testing.T, mr *miniredis.Miniredis, rc *redis.Client) int {
	t.Helper()
	processed := 0
	for i := 1; i <= maxDeliveries; i++ {
		mr.SetTime(base.Add(time.Duration(i) * (ClaimMinIdle + time.Second)))
		err := ClaimPending(t.Context(), rc, testStream, testGroup, nextWorker,
			func(context.Context, string, map[string]any) error {
				processed++
				return errors.New("still down")
			})
		if err != nil {
			t.Fatalf("ClaimPending (%d 周目): %v", i, err)
		}
	}
	return processed
}

func TestClaimPendingDeadLettersAtMaxDeliveries(t *testing.T) {
	mr, rc := newPending(t, settled)

	processed := claimUntilDLQ(t, mr, rc)

	// 1 回目の配送は newPending の XReadGroup が消費している。
	if want := maxDeliveries - 1; processed != want {
		t.Fatalf("processed = %d, want %d", processed, want)
	}
	if n := pendingCount(t, rc); n != 0 {
		t.Fatalf("pending = %d, want 0 (DLQ へ退避して ack される)", n)
	}
}

func TestClaimPendingDrainsBeyondOneBatch(t *testing.T) {
	const total = 20
	values := make([]map[string]any, 0, total)
	for i := range total {
		values = append(values, map[string]any{"event": "payment.settled", "orderId": fmt.Sprint(i)})
	}
	mr, rc := newPending(t, values...)
	mr.SetTime(base.Add(ClaimMinIdle + time.Second))

	processed := 0
	err := ClaimPending(t.Context(), rc, testStream, testGroup, nextWorker,
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

func TestClaimPendingStopsOnCanceledContext(t *testing.T) {
	mr, rc := newPending(t, settled)
	mr.SetTime(base.Add(ClaimMinIdle + time.Second))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := ClaimPending(ctx, rc, testStream, testGroup, nextWorker,
		func(_ context.Context, _ string, _ map[string]any) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ClaimPending() = %v, want context.Canceled", err)
	}
}
