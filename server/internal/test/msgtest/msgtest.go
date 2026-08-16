// Package msgtest は messaging のポートのテスト用実装を共有する。
package msgtest

import (
	"context"

	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/strconvx"
)

// Subscriber は配信済みメッセージを 1 度だけ返す購読。messaging.Consume は無限ループなので、
// 配り終えたら stop を呼んでループを終わらせる。
type Subscriber struct {
	Topic  string
	Queue  string
	Acked  []string
	values []map[string]any
	stop   context.CancelFunc
	drawn  bool
}

func NewSubscriber(stop context.CancelFunc, values ...map[string]any) *Subscriber {
	return &Subscriber{stop: stop, values: values}
}

func (s *Subscriber) Subscribe(_ context.Context, topic, queue string) (messaging.Subscription, error) {
	s.Topic, s.Queue = topic, queue
	return s, nil
}

func (s *Subscriber) Receive(_ context.Context) ([]messaging.Received, error) {
	if s.drawn {
		s.stop()
		return nil, nil
	}
	s.drawn = true
	received := make([]messaging.Received, 0, len(s.values))
	for i, v := range s.values {
		received = append(received, messaging.Received{Handle: handle(i), Values: v})
	}
	return received, nil
}

func (s *Subscriber) Ack(_ context.Context, handle string) error {
	s.Acked = append(s.Acked, handle)
	return nil
}

func handle(i int) string {
	return strconvx.FormatInt64(int64(i))
}
