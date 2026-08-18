package consumer

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
)

// 滞留と再配送を settled 受信と独立させ、片方の詰まりが他方を止めない。
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

// (ADR-[[202606250159]])
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
	ev, err := orderevent.ParseCancelled(values)
	if errors.Is(err, orderevent.ErrNotCancelled) {
		return nil
	}
	if err != nil {
		// (ADR-[[202608150830]])
		slog.ErrorContext(ctx, "shipping cancel consumer: invalid payload", "error", err)
		return err
	}
	return c.canceller.CancelShipmentForOrder(ctx, ev.OrderID)
}
