// Package consumer は決済確定イベントを購読して配送枠を作る (ADR-[[202606211200]])。
package consumer

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
	"github.com/rin2yh/study-architecture/server/shipping/internal/gateway"
)

const queue = "payment-events-shipping"

// メッセージごとに引かないよう保持する。otel の global は遅延差し替えに対応するので、
// TracerProvider 設定前に取得しても問題ない。
var tracer = otel.Tracer("shipping-worker")

type ShipmentCreator interface {
	CreateShipmentForOrder(ctx context.Context, orderID order.ID, dest gateway.Destination) (db.ShippingShipment, error)
}

type Consumer struct {
	subscriber messaging.Subscriber
	creator    ShipmentCreator
	order      gateway.OrderPort
}

func New(subscriber messaging.Subscriber, creator ShipmentCreator, order gateway.OrderPort) *Consumer {
	return &Consumer{subscriber: subscriber, creator: creator, order: order}
}

func (c *Consumer) Run(ctx context.Context) error {
	sub, err := c.subscriber.Subscribe(ctx, paymentevent.Topic, queue)
	if err != nil {
		return err
	}
	return messaging.Consume(ctx, queue, sub, c.process)
}

// producer の発行 trace とは親子でなく link で結ぶ (ADR-[[202606250159]])。
func (c *Consumer) process(ctx context.Context, values map[string]any) error {
	ctx, span := tracer.Start(ctx, "payment.settled process",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(paymentevent.LinkFrom(ctx, values)),
	)
	defer span.End()

	err := c.handle(ctx, values)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// span が有効なうちに記録し、ログを trace と相関させる。
		slog.ErrorContext(ctx, "shipping consumer: handle failed", "error", err)
	}
	return err
}

func (c *Consumer) handle(ctx context.Context, values map[string]any) error {
	ev, err := paymentevent.ParseSettled(values)
	if errors.Is(err, paymentevent.ErrNotSettled) {
		return nil
	}
	if err != nil {
		// 再配送しても直らない payload は上限超過でブローカが DLQ へ隔離する (ADR-[[202608150830]])。
		slog.ErrorContext(ctx, "shipping consumer: invalid payload", "error", err)
		return err
	}
	// (ADR-[[202606301000]])
	dest, err := c.order.FetchDestination(ctx, ev.OrderID)
	if err != nil {
		return err
	}
	_, err = c.creator.CreateShipmentForOrder(ctx, ev.OrderID, dest)
	if errors.Is(err, dberr.ErrConflict) {
		return nil
	}
	return err
}
