package redisx

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 処理中の consumer からメッセージを奪わないための猶予。ハンドラが内包する同期呼び出しの
	// 最悪ケース (resilience の AttemptTimeout 3s × MaxAttempts 3 + バックオフ ≒ 10s) より
	// 十分長く取る。poison message の再試行間隔もこの値で決まる (上限と隔離は #106)。
	claimMinIdle = 30 * time.Second
	// 1 回の XAUTOCLAIM で走査する PEL 件数。新規読み (XReadGroup の Count) と同じ粒度にそろえる。
	claimCount = 16
	// XAUTOCLAIM は走査を完走するとこの値をカーソルとして返すので、終了判定にも使う。
	pelStart = "0-0"
)

// ProcessFunc は引き取ったメッセージ 1 件の処理。nil を返した分だけ ack される。
type ProcessFunc func(ctx context.Context, id string, values map[string]any) error

// ClaimPending は min-idle を過ぎた未 ACK メッセージを consumer 名義で引き取り、process で再処理する。
// XReadGroup の ">" は PEL を返さないため、これを回さない限り一度失敗したメッセージは再処理されない。
func ClaimPending(ctx context.Context, rdb *redis.Client, stream, group, consumer string, process ProcessFunc) error {
	start := pelStart
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		msgs, next, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumer,
			MinIdle:  claimMinIdle,
			Start:    start,
			Count:    claimCount,
		}).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(msgs) > 0 {
			// 引き取りが起きた = 過去に処理が失敗している。件数を可視化しておく。
			slog.Info("redisx: claimed pending messages", "stream", stream, "group", group, "count", len(msgs))
		}
		for _, m := range msgs {
			if err := process(ctx, m.ID, m.Values); err != nil {
				// process 内で記録済み。ack せず PEL に残し、次の min-idle 経過後に再度引き取る。
				continue
			}
			if err := rdb.XAck(ctx, stream, group, m.ID).Err(); err != nil {
				slog.Warn("redisx: xack failed", "stream", stream, "group", group, "id", m.ID, "error", err)
			}
		}
		// 引き取った分は idle が 0 に戻るため、走査を終えたら次の周回に委ねる。カーソルが
		// 進まない実装に当たっても抜けられるよう start との一致も終了条件に入れる。
		if next == pelStart || next == "" || next == start {
			return nil
		}
		start = next
	}
}
