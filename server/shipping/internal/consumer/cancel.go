package consumer

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
)

// settled 受信とは別トピック・別キュー。滞留と再配送を独立させ、片方の詰まりが他方を止めない。
const cancelQueue = "order-events-shipping"

type ShipmentCanceller interface {
	CancelShipmentForOrder(ctx context.Context, orderID order.ID) error
}

type CancelConsumer struct {
	subscriber messaging.Subscriber
	canceller  ShipmentCanceller
}

func NewCancel(subscriber messaging.Subscriber, canceller ShipmentCanceller) *CancelConsumer {
	return &CancelConsumer{subscriber: subscriber, canceller: canceller}
}

func (c *CancelConsumer) Run(ctx context.Context) error {
	sub, err := c.subscriber.Subscribe(ctx, orderevent.Topic, cancelQueue)
	if err != nil {
		return err
	}
	return messaging.Consume(ctx, cancelQueue, sub, c.process)
}

// producer の発行 trace とは親子でなく link で結ぶ (ADR-[[202606250159]])。
func (c *CancelConsumer) process(ctx context.Context, values map[string]any) error {
	ctx, span := tracer.Start(ctx, "order.cancelled process",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(orderevent.LinkFrom(ctx, values)),
	)
	defer span.End()

	err := c.handle(ctx, values)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "shipping cancel consumer: handle failed", "error", err)
	}
	return err
}

func (c *CancelConsumer) handle(ctx context.Context, values map[string]any) error {
	if t, _ := values[orderevent.FieldEvent].(string); t != orderevent.TypeCancelled {
		return nil
	}
	orderID, err := order.ParseIDFromEvent(values)
	if err != nil {
		// 再配送しても直らない payload は上限超過でブローカが DLQ へ隔離する (ADR-[[202608150830]])。
		slog.ErrorContext(ctx, "shipping cancel consumer: invalid orderId", "error", err)
		return err
	}
	return c.canceller.CancelShipmentForOrder(ctx, orderID)
}
