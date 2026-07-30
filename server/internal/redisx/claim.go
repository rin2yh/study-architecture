package redisx

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// ClaimMinIdle は処理中の consumer からメッセージを奪わないための猶予。ハンドラが内包する
// 同期呼び出しの最悪ケース (resilience の AttemptTimeout 3s × MaxAttempts 3 + バックオフ ≒ 10s)
// より十分長く取る。poison message の再試行間隔もこの値で決まる (上限と隔離は #106)。
const ClaimMinIdle = 30 * time.Second

const (
	// 1 回の走査で引き取る上限。1 周回の処理量を新規読み (XReadGroup の Count) と同程度に抑える。
	claimCount = 16
	// 1 呼び出しあたりの走査回数の上限。PEL が肥大しても周回が長時間ブロックしないようにする。
	claimPasses = 4
	// XAUTOCLAIM は走査を完走するとこの値をカーソルとして返すので、終了判定にも使う。
	pelStart = "0-0"
)

// ClaimPending は min-idle を過ぎた未 ACK メッセージを consumer 名義で引き取り、process で
// 再処理する (nil を返した分だけ ack する)。XReadGroup の ">" は PEL を返さないため、これを
// 回さない限り一度失敗したメッセージは再処理されない。
func ClaimPending(ctx context.Context, rdb *redis.Client, stream, group, consumer string, process func(ctx context.Context, id string, values map[string]any) error) error {
	start := pelStart
	for range claimPasses {
		if err := ctx.Err(); err != nil {
			return err
		}
		msgs, next, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumer,
			MinIdle:  ClaimMinIdle,
			Start:    start,
			Count:    claimCount,
		}).Result()
		if err != nil {
			return err
		}
		if len(msgs) > 0 {
			// 引き取りが起きた = 過去に処理が失敗している。
			slog.Info("redisx: claimed pending messages", "stream", stream, "group", group, "count", len(msgs))
		}
		acked := make([]string, 0, len(msgs))
		for _, m := range msgs {
			if err := process(ctx, m.ID, m.Values); err != nil {
				// process 内で記録済み。ack せず PEL に残し、次の min-idle 経過後に再度引き取る。
				continue
			}
			acked = append(acked, m.ID)
		}
		if len(acked) > 0 {
			if err := rdb.XAck(ctx, stream, group, acked...).Err(); err != nil {
				slog.Warn("redisx: xack failed", "stream", stream, "group", group, "ids", acked, "error", err)
			}
		}
		if next == pelStart {
			return nil
		}
		start = next
	}
	// 引き取った分は idle が 0 に戻るため、走査しきれなかった残りは次の周回に委ねる。
	return nil
}
