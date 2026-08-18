// Package consumer は決済確定イベントを購読して在庫予約を確定する。
package consumer

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
)

const queue = "payment-events-inventory"

var tracer = otel.Tracer("inventory-worker")

type ReservationConfirmer interface {
	ConfirmReservationsByOrder(ctx context.Context, orderID order.ID) error
}

type Consumer struct {
	subscriber messaging.Subscriber
	confirmer  ReservationConfirmer
}

func New(subscriber messaging.Subscriber, confirmer ReservationConfirmer) *Consumer {
	return &Consumer{subscriber: subscriber, confirmer: confirmer}
}

func (c *Consumer) Run(ctx context.Context) error {
	sub, err := c.subscriber.Subscribe(ctx, paymentevent.Topic, queue)
	if err != nil {
		return err
	}
	return messaging.Consume(ctx, queue, sub, c.process)
}

// (ADR-[[202606250159]])
func (c *Consumer) process(ctx context.Context, values map[string]any) error {
	ctx, span := tracer.Start(ctx, "payment.settled confirm",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(paymentevent.LinkFrom(ctx, values)),
	)
	defer span.End()

	err := c.handle(ctx, values)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "inventory consumer: handle failed", "error", err)
	}
	return err
}

func (c *Consumer) handle(ctx context.Context, values map[string]any) error {
	ev, err := paymentevent.ParseSettled(values)
	if errors.Is(err, paymentevent.ErrNotSettled) {
		return nil
	}
	if err != nil {
		// (ADR-[[202608150830]])
		slog.ErrorContext(ctx, "inventory consumer: invalid payload", "error", err)
		return err
	}
	// (ADR-[[202606261214]])
	return c.confirmer.ConfirmReservationsByOrder(ctx, ev.OrderID)
}
