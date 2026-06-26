// Package consumer は決済確定イベントを購読して予約を確定へ昇格する。
package consumer

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
)

const (
	consumerGroup = "inventory"
	backoff       = time.Second
)

var tracer = otel.Tracer("inventory-worker")

type ReservationConfirmer interface {
	ConfirmReservationsByOrder(ctx context.Context, orderID int64) error
}

type Consumer struct {
	rdb       *redis.Client
	confirmer ReservationConfirmer
	name      string
	block     time.Duration
}

func New(rc *redis.Client, confirmer ReservationConfirmer) *Consumer {
	// 再起動後も pending を引き取れるよう、識別名はランダムでなく安定値 (hostname) にする。
	name, _ := os.Hostname()
	if name == "" {
		name = consumerGroup
	}
	return &Consumer{rdb: rc, confirmer: confirmer, name: name, block: 5 * time.Second}
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	slog.Info("inventory consumer started", "stream", paymentevent.Stream, "group", consumerGroup, "consumer", c.name)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.readAndProcess(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Warn("inventory consumer: read failed, backing off", "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
}

func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, paymentevent.Stream, consumerGroup, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (c *Consumer) readAndProcess(ctx context.Context) error {
	res, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: c.name,
		Streams:  []string{paymentevent.Stream, ">"},
		Count:    16,
		Block:    c.block,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, st := range res {
		for _, m := range st.Messages {
			if err := c.process(ctx, m.ID, m.Values); err != nil {
				continue
			}
			if err := c.rdb.XAck(ctx, paymentevent.Stream, consumerGroup, m.ID).Err(); err != nil {
				slog.Warn("inventory consumer: xack failed", "id", m.ID, "error", err)
			}
		}
	}
	return nil
}

// producer の発行 trace とは親子でなく link で結ぶ (ADR-[[202606250159]])。
func (c *Consumer) process(ctx context.Context, id string, values map[string]any) error {
	ctx, span := tracer.Start(ctx, "payment.settled confirm",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(paymentevent.LinkFrom(ctx, values)),
	)
	defer span.End()

	err := c.handle(ctx, values)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "inventory consumer: handle failed", "id", id, "error", err)
	}
	return err
}

func (c *Consumer) handle(ctx context.Context, values map[string]any) error {
	if t, _ := values[paymentevent.FieldEvent].(string); t != paymentevent.TypeSettled {
		return nil
	}
	raw, _ := values[paymentevent.FieldOrderID].(string)
	orderID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		// 壊れた payload は再配送しても直らない。pending を膨らませないため握って可視化のみ。
		slog.ErrorContext(ctx, "inventory consumer: invalid orderId", "raw", raw, "error", err)
		return nil
	}
	// 確定は ON CONFLICT DO NOTHING で冪等。再配信は no-op で ack される (ADR-[[202606261214]])。
	return c.confirmer.ConfirmReservationsByOrder(ctx, orderID)
}
