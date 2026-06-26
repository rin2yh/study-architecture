// Package outbox は Transactional Outbox のリレーを共有実装する (ADR-[[202606261212]])。
// 送信状態を自DBの集約に持つサービスが、未送信行をポーリングして Redis Streams へ送出する
// ループをここに 1 つ置く。発行サービスが増えても各自のプロセス内でこれを回すだけでよい。
package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Message は 1 件の未送信イベント。Values がそのまま XAdd のペイロードで、trace 伝播フィールドは
// Store が Values に載せて返す (リレーはイベントの中身を知らない)。
type Message struct {
	ID     int64
	Stream string
	Values map[string]any
}

// Store は未送信イベントの取得と送信済みマークを担う。実装は各サービスが自DBに対して用意する。
type Store interface {
	FetchUnpublished(ctx context.Context, limit int) ([]Message, error)
	MarkPublished(ctx context.Context, id int64) error
}

type Relay struct {
	rdb      *redis.Client
	store    Store
	interval time.Duration
	batch    int
}

func NewRelay(rdb *redis.Client, store Store) *Relay {
	return &Relay{rdb: rdb, store: store, interval: time.Second, batch: 64}
}

func (r *Relay) Run(ctx context.Context) error {
	slog.Info("outbox relay started", "interval", r.interval)
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		// 再起動で取り残された未送信を interval 待ちなしで送るため、tick より先に流す。
		if err := r.drain(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// 送出失敗は行を pending のまま残し次の tick で再送する (at-least-once)。可視化だけする。
			slog.Warn("outbox relay: drain failed, will retry next tick", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}

func (r *Relay) drain(ctx context.Context) error {
	for {
		msgs, err := r.store.FetchUnpublished(ctx, r.batch)
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if err := r.rdb.XAdd(ctx, &redis.XAddArgs{Stream: m.Stream, Values: m.Values}).Err(); err != nil {
				return err
			}
			// XAdd 成功後に落ちると同じ行を次回また送るが、受信側の冪等性 (ADR-[[202606261214]]) で吸収する。
			if err := r.store.MarkPublished(ctx, m.ID); err != nil {
				return err
			}
		}
		// 滞留時に送出が tick 間隔ぶん遅れないよう、未送信が尽きるまで続けて引く。
		if len(msgs) < r.batch {
			return nil
		}
	}
}
