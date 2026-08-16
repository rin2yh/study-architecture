// Package consumer は注文キャンセルイベントを購読して返金 (補償) を行う。
// フォワードの payment.settled 発行 (outbox) と対称な逆流側の入口 (ADR-[[202606261702]])。
package consumer

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
)

const queue = "order-events-payment"

var tracer = otel.Tracer("payment")

type PaymentRefunder interface {
	RefundByOrder(ctx context.Context, orderID order.ID) error
}

type Consumer struct {
	subscriber messaging.Subscriber
	refunder   PaymentRefunder
}

func New(subscriber messaging.Subscriber, refunder PaymentRefunder) *Consumer {
	return &Consumer{subscriber: subscriber, refunder: refunder}
}

func (c *Consumer) Run(ctx context.Context) error {
	sub, err := c.subscriber.Subscribe(ctx, orderevent.Topic, queue)
	if err != nil {
		return err
	}
	return messaging.Consume(ctx, queue, sub, c.process)
}

// producer の発行 trace とは親子でなく link で結ぶ (ADR-[[202606250159]])。
func (c *Consumer) process(ctx context.Context, values map[string]any) error {
	ctx, span := tracer.Start(ctx, "order.cancelled refund",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(orderevent.LinkFrom(ctx, values)),
	)
	defer span.End()

	err := c.handle(ctx, values)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "payment consumer: handle failed", "error", err)
	}
	return err
}

func (c *Consumer) handle(ctx context.Context, values map[string]any) error {
	if t, _ := values[orderevent.FieldEvent].(string); t != orderevent.TypeCancelled {
		return nil
	}
	orderID, err := order.ParseIDFromEvent(values)
	if err != nil {
		// 再配送しても直らない payload は上限超過でブローカが DLQ へ隔離する (ADR-[[202608150830]])。
		slog.ErrorContext(ctx, "payment consumer: invalid orderId", "error", err)
		return err
	}
	return c.refunder.RefundByOrder(ctx, orderID)
}
