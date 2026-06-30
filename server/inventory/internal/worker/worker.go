// Package worker は payment.settled 購読 (consumer)・order.cancelled 購読 (cancel)・TTL 回収 (reaper) を
// 1 プロセスで束ねる。いずれも同じ DB プールを共有するため 1 つの DI グラフでまとめ、二重接続を避ける。
package worker

import (
	"context"

	"github.com/rin2yh/study-architecture/server/inventory/internal/consumer"
	"github.com/rin2yh/study-architecture/server/inventory/internal/reaper"
)

type Worker struct {
	consumer *consumer.Consumer
	cancel   *consumer.CancelConsumer
	reaper   *reaper.Reaper
}

func New(c *consumer.Consumer, cancel *consumer.CancelConsumer, r *reaper.Reaper) *Worker {
	return &Worker{consumer: c, cancel: cancel, reaper: r}
}

// Run は全 goroutine を並行に回し、どれかが終了したら他も止めて最初のエラーを返す。
// 1 つが静かに死んでも fail loud で worker ごと落とし再起動に委ねる (ADR-[[202606211200]])。
func (w *Worker) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, 3)
	go func() { errc <- w.consumer.Run(ctx) }()
	go func() { errc <- w.cancel.Run(ctx) }()
	go func() { errc <- w.reaper.Run(ctx) }()

	err := <-errc
	cancel()
	<-errc
	<-errc
	return err
}
