// Package worker は payment.settled 購読 (consumer) と TTL 回収 (reaper) を 1 プロセスで束ねる。
// 両者は同じ DB プールを共有するため 1 つの DI グラフでまとめ、二重接続を避ける。
package worker

import (
	"context"

	"github.com/rin2yh/study-architecture/server/inventory/internal/consumer"
	"github.com/rin2yh/study-architecture/server/inventory/internal/reaper"
)

type Worker struct {
	consumer *consumer.Consumer
	reaper   *reaper.Reaper
}

func New(c *consumer.Consumer, r *reaper.Reaper) *Worker {
	return &Worker{consumer: c, reaper: r}
}

// Run は consumer と reaper を並行に回し、どちらかが終了したら他方も止めて最初のエラーを返す。
// 片方が静かに死んでも fail loud で worker ごと落とし再起動に委ねる (ADR-[[202606211200]])。
func (w *Worker) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, 2)
	go func() { errc <- w.consumer.Run(ctx) }()
	go func() { errc <- w.reaper.Run(ctx) }()

	err := <-errc
	cancel()
	<-errc
	return err
}
