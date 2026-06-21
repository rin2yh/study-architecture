// Package event は payment が確定などのドメインイベントを broker (Redis Streams) へ
// publish する出力ポートと実装をまとめる。配送手配の起点 (ADR-[[202606211200]])。
package event

import (
	"context"
	"errors"
	"os"

	"github.com/redis/go-redis/v9"
)

const (
	streamPaymentEvents = "payment.events"
	typePaymentSettled  = "payment.settled"
)

type PaymentSettled struct {
	PaymentID   int64
	OrderID     int64
	AmountCents int64
}

type Publisher interface {
	PublishPaymentSettled(ctx context.Context, e PaymentSettled) error
}

// Settled は status が「決済確定」を表すかを判定する。確定の語彙はサービス間で揺れるため
// (capture/settle/paid 相当)、配送手配のトリガとなる確定状態をここで一元的に定義する。
func Settled(status string) bool {
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
	url := os.Getenv("REDIS_URL")
	if url == "" {
		return nil, errors.New("REDIS_URL is required")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &RedisPublisher{rdb: redis.NewClient(opt)}, nil
}

func (p *RedisPublisher) PublishPaymentSettled(ctx context.Context, e PaymentSettled) error {
	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamPaymentEvents,
		Values: map[string]any{
			"event":       typePaymentSettled,
			"paymentId":   e.PaymentID,
			"orderId":     e.OrderID,
			"amountCents": e.AmountCents,
		},
	}).Err()
}
