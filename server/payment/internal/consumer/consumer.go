// Package consumer は注文キャンセルイベントを購読して返金 (補償) を行う。
// フォワードの payment.settled 発行 (outbox) と対称な逆流側の入口 (ADR-[[202606261702]])。
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

	"github.com/rin2yh/study-architecture/server/internal/orderevent"
)

const (
	consumerGroup = "payment"
	backoff       = time.Second
)

var tracer = otel.Tracer("payment")

type PaymentRefunder interface {
	RefundByOrder(ctx context.Context, orderID int64) error
}

type Consumer struct {
	rdb      *redis.Client
	refunder PaymentRefunder
	name     string
	block    time.Duration
}

func New(rc *redis.Client, refunder PaymentRefunder) *Consumer {
	// 再起動後も pending を引き取れるよう、識別名はランダムでなく安定値 (hostname) にする。
	name, _ := os.Hostname()
	if name == "" {
		name = consumerGroup
	}
	return &Consumer{rdb: rc, refunder: refunder, name: name, block: 5 * time.Second}
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	slog.Info("payment consumer started", "stream", orderevent.Stream, "group", consumerGroup, "consumer", c.name)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.readAndProcess(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Warn("payment consumer: read failed, backing off", "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
}

func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, orderevent.Stream, consumerGroup, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (c *Consumer) readAndProcess(ctx context.Context) error {
	res, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: c.name,
		Streams:  []string{orderevent.Stream, ">"},
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
			if err := c.rdb.XAck(ctx, orderevent.Stream, consumerGroup, m.ID).Err(); err != nil {
				slog.Warn("payment consumer: xack failed", "id", m.ID, "error", err)
			}
		}
	}
	return nil
}

// producer の発行 trace とは親子でなく link で結ぶ (ADR-[[202606250159]])。
func (c *Consumer) process(ctx context.Context, id string, values map[string]any) error {
	ctx, span := tracer.Start(ctx, "order.cancelled refund",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(orderevent.LinkFrom(ctx, values)),
	)
	defer span.End()

	err := c.handle(ctx, values)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "payment consumer: handle failed", "id", id, "error", err)
	}
	return err
}

func (c *Consumer) handle(ctx context.Context, values map[string]any) error {
	if t, _ := values[orderevent.FieldEvent].(string); t != orderevent.TypeCancelled {
		return nil
	}
	raw, _ := values[orderevent.FieldOrderID].(string)
	orderID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		// 壊れた payload は再配送しても直らない。pending を膨らませないため握って可視化のみ。
		slog.ErrorContext(ctx, "payment consumer: invalid orderId", "raw", raw, "error", err)
		return nil
	}
	return c.refunder.RefundByOrder(ctx, orderID)
}
