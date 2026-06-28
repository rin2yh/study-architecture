// Package worker は payment.settled 購読 (配送手配) と order.cancelled 購読 (配送中止) を
// 1 プロセスで束ねる。両者は同じ DB プールを共有するため 1 つの DI グラフでまとめ、二重接続を避ける。
package worker

import (
	"context"

	"github.com/rin2yh/study-architecture/server/shipping/internal/consumer"
)

type Worker struct {
	settled *consumer.Consumer
	cancel  *consumer.CancelConsumer
}

func New(settled *consumer.Consumer, cancel *consumer.CancelConsumer) *Worker {
	return &Worker{settled: settled, cancel: cancel}
}

// Run は両 consumer を並行に回し、どちらかが終了したら他方も止めて最初のエラーを返す。
// 片方が静かに死んでも fail loud で worker ごと落とし再起動に委ねる (ADR-[[202606211200]])。
func (w *Worker) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, 2)
	go func() { errc <- w.settled.Run(ctx) }()
	go func() { errc <- w.cancel.Run(ctx) }()

	err := <-errc
	cancel()
	<-errc
	return err
}
