package redisx

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// ClaimMinIdle は処理中の consumer からメッセージを奪わないための猶予。ハンドラが内包する
// 同期呼び出しの最悪ケース (resilience の AttemptTimeout 3s × MaxAttempts 3 + バックオフ ≒ 10s)
// より十分長く取る。恒久的に失敗するメッセージの再試行間隔もこの値で決まる。
const ClaimMinIdle = 30 * time.Second

const (
	// 1 回の走査で引き取る上限。1 周回の処理量を新規読み (XReadGroup の Count) と同程度に抑える。
	claimCount = 16
	// 1 呼び出しあたりの走査回数の上限。PEL が肥大しても周回が長時間ブロックしないようにする。
	claimPasses = 4
	pelStart    = "-"
	pelEnd      = "+"
)

// ClaimPending は min-idle を過ぎた未 ACK メッセージを consumer 名義で引き取り、process で
// 再処理する (nil を返した分だけ ack する)。XReadGroup の ">" は PEL を返さないため、これを
// 回さない限り一度失敗したメッセージは再処理されない。
// 配送回数が MaxDeliveries に達したものは process へ通さず DLQ へ退避して ack する
// (ADR-[[202607301418]])。
func ClaimPending(ctx context.Context, rdb *redis.Client, stream, group, consumer string, process func(ctx context.Context, id string, values map[string]any) error) error {
	for range claimPasses {
		if err := ctx.Err(); err != nil {
			return err
		}
		// 配送回数で仕分けるため PEL を先に読む。XAUTOCLAIM は値を返す代わりに配送回数を返さない。
		pending, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: stream,
			Group:  group,
			Idle:   ClaimMinIdle,
			Start:  pelStart,
			End:    pelEnd,
			Count:  claimCount,
		}).Result()
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}
		deliveries := make(map[string]int64, len(pending))
		ids := make([]string, 0, len(pending))
		for _, p := range pending {
			deliveries[p.ID] = p.RetryCount
			ids = append(ids, p.ID)
		}
		// min-idle を再指定して、XPENDING との間に他の consumer が引き取った分は掴まないようにする。
		msgs, err := rdb.XClaim(ctx, &redis.XClaimArgs{
			Stream:   stream,
			Group:    group,
			Consumer: consumer,
			MinIdle:  ClaimMinIdle,
			Messages: ids,
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
			if deliveries[m.ID] >= MaxDeliveries {
				if err := deadLetter(ctx, rdb, stream, group, m, deliveries[m.ID]); err != nil {
					slog.Error("redisx: dead letter failed", "stream", stream, "group", group, "id", m.ID, "error", err)
					continue
				}
				slog.Error("redisx: message moved to dlq", "stream", stream, "group", group, "id", m.ID, "deliveries", deliveries[m.ID])
				acked = append(acked, m.ID)
				continue
			}
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
	}
	// 引き取った分は idle が 0 に戻り次の XPENDING に載らないので、走査しきれなかった残りは
	// 次の周回に委ねる。
	return nil
}
