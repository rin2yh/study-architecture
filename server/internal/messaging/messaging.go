// Package messaging は非同期イベントの発行と購読をブローカ非依存の形で定める。
// 実装 (SQS / Pub/Sub) はアダプタ側に置き、ハンドラにベンダ API を見せない (ADR-[[202608150835]])。
package messaging

import (
	"context"
	"log/slog"
)

// Publisher は発行側の出力ポート。宛先はトピック名で指定する。
type Publisher interface {
	Publish(ctx context.Context, topic string, values map[string]any) error
}

// Received は購読側が受け取った 1 件。Handle は Ack に渡す不透明なハンドル。
type Received struct {
	Handle string
	Values map[string]any
}

// Subscription は 1 つのキューへの購読。Ack しなかった分はブローカが再配送し、
// 上限を超えたら DLQ へ隔離する (ADR-[[202608150830]])。
type Subscription interface {
	Receive(ctx context.Context) ([]Received, error)
	Ack(ctx context.Context, handle string) error
}

// Subscriber は購読側の入力ポート。topic に対する queue を用意して購読を返す。
type Subscriber interface {
	Subscribe(ctx context.Context, topic, queue string) (Subscription, error)
}

// Consume は購読の受信ループ。process が nil を返した分だけ Ack する。
// 失敗は Ack しないことで再配送に委ねるため、ここで再試行の間隔や回数は持たない。
func Consume(ctx context.Context, name string, sub Subscription, process func(ctx context.Context, values map[string]any) error) error {
	slog.Info("consumer started", "consumer", name)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		msgs, err := sub.Receive(ctx)
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if err := process(ctx, m.Values); err != nil {
				continue
			}
			if err := sub.Ack(ctx, m.Handle); err != nil {
				// Ack 漏れは再配送されるが、受信側は冪等 (ADR-[[202606261214]])。
				slog.Warn("consumer: ack failed", "consumer", name, "error", err)
			}
		}
	}
}
