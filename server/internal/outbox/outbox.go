// Package outbox は Transactional Outbox のリレーを共有実装する (ADR-[[202606261212]])。
// 送信状態を自DBの集約に持つサービスが、未送信行をポーリングして Redis Streams へ送出する
// ループをここに 1 つ置く。発行サービスが増えても各自のプロセス内でこれを回すだけでよい。
package outbox

import (
	"bytes"
	"context"
	"encoding/json"
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

// W3C traceparent の Values キー。発行と消費を span link でつなぐためリレー層が送出に載せ直す
// (ADR-[[202606250159]])。
const FieldTraceparent = "traceparent"

// Event は outbox に積めるドメインイベント。発行に必要な宛先 (種別・集約) と中身を自分で知っており、
// producer は「起きた事実」をこの形で Dispatch に渡すだけでよい。
type Event interface {
	EventType() string
	AggregateID() int64
	Values() map[string]any
}

// Row は outbox 1 行ぶんの INSERT 値。各サービスの sqlc 実装をこの形へ薄く適合させる
// (生成型がパッケージごとに違うため、共有層はこの中立な型で受ける)。
type Row struct {
	AggregateID int64
	EventType   string
	Payload     []byte
	Traceparent string
}

// Inserter は呼び出し元の tx に紐づく Querier を包み、outbox 行を 1 件 INSERT する。
type Inserter interface {
	InsertOutbox(ctx context.Context, row Row) error
}

// Dispatch は発行イベントを payload(jsonb) 化し、業務更新と同一 tx の outbox へ積む。各 command が
// marshal と INSERT を手書きしないための単一の発行口で、producer や発行イベント種が増えても
// ここを通すだけで「commit 済みは必ず送出」の保証を継承できる。
func Dispatch(ctx context.Context, ins Inserter, traceparent string, events ...Event) error {
	for _, ev := range events {
		payload, err := json.Marshal(ev.Values())
		if err != nil {
			return err
		}
		if err := ins.InsertOutbox(ctx, Row{
			AggregateID: ev.AggregateID(),
			EventType:   ev.EventType(),
			Payload:     payload,
			Traceparent: traceparent,
		}); err != nil {
			return err
		}
	}
	return nil
}

// DecodePayload は outbox 行の payload (jsonb) と traceparent を XAdd 用の Values へ復元する。
// Dispatch が載せた Values をそのまま戻すだけでよく、リレーはイベントの中身を知らないままにできる。
// bigint が JSON 経由で float64 化して桁落ちしないよう、整数は json.Number から int64 へ戻す。
func DecodePayload(raw []byte, traceparent string) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var values map[string]any
	if err := dec.Decode(&values); err != nil {
		return nil, err
	}
	for k, v := range values {
		n, ok := v.(json.Number)
		if !ok {
			continue
		}
		if i, err := n.Int64(); err == nil {
			values[k] = i
		} else {
			values[k] = n.String()
		}
	}
	// consumer 側の span link を切らさないため (ADR-[[202606250159]])。
	if traceparent != "" {
		values[FieldTraceparent] = traceparent
	}
	return values, nil
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
