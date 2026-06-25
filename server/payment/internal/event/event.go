// Package event は payment のドメインイベントの publish を担う。
package event

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/internal/redisx"
)

type Publisher interface {
	PublishPaymentSettled(ctx context.Context, e paymentevent.Settled) error
}

// IsSettled は status が「決済確定」を表すかを判定する。確定の語彙はサービス間で揺れるため
// (capture/settle/paid 相当)、配送手配のトリガとなる確定状態をここで一元的に定義する。
func IsSettled(status string) bool {
	switch status {
	case "paid", "settled", "captured":
		return true
	default:
		return false
	}
}

type RedisPublisher struct {
	rdb *redis.Client
}

var _ Publisher = (*RedisPublisher)(nil)

func NewRedisPublisher() (*RedisPublisher, error) {
	rc, err := redisx.NewClient()
	if err != nil {
		return nil, err
	}
	return &RedisPublisher{rdb: rc}, nil
}

func (p *RedisPublisher) PublishPaymentSettled(ctx context.Context, e paymentevent.Settled) error {
	values := e.Values()
	paymentevent.Inject(ctx, values)
	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: paymentevent.Stream,
		Values: values,
	}).Err()
}
