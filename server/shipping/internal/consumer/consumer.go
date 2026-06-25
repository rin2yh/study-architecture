// Package consumer は決済確定イベントを購読して配送を手配する。
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

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

const (
	consumerGroup = "shipping"
	backoff       = time.Second
)

// メッセージごとに引かないよう保持する。otel の global は遅延差し替えに対応するので、
// TracerProvider 設定前に取得しても問題ない。
var tracer = otel.Tracer("shipping-worker")

type ShipmentCreator interface {
	CreateShipmentForOrder(ctx context.Context, orderID int64) (db.ShippingShipment, error)
}

type Consumer struct {
	rdb     *redis.Client
	creator ShipmentCreator
	name    string
	block   time.Duration
}

func New(rc *redis.Client, creator ShipmentCreator) *Consumer {
	// 同一グループ内で consumer を識別する名前。再起動後も pending を引き取れるよう
	// ランダムでなく安定値 (hostname) にする。
	name, _ := os.Hostname()
	if name == "" {
		name = consumerGroup
	}
	return &Consumer{rdb: rc, creator: creator, name: name, block: 5 * time.Second}
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	slog.Info("shipping consumer started", "stream", paymentevent.Stream, "group", consumerGroup, "consumer", c.name)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.readAndProcess(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			slog.Warn("shipping consumer: read failed, backing off", "error", err)
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
	// 2 回目以降の起動では group が既にあり BUSYGROUP になるが、これは正常。
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
				// span 内で記録済み。ack せず pending に残し、次回 (XReadGroup の再配送) に委ねる。
				continue
			}
			if err := c.rdb.XAck(ctx, paymentevent.Stream, consumerGroup, m.ID).Err(); err != nil {
				slog.Warn("shipping consumer: xack failed", "id", m.ID, "error", err)
			}
		}
	}
	return nil
}

// producer の発行 trace とは親子でなく link で結ぶ (ADR-[[202606250159]])。
func (c *Consumer) process(ctx context.Context, id string, values map[string]any) error {
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
		slog.ErrorContext(ctx, "shipping consumer: handle failed", "id", id, "error", err)
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
		slog.ErrorContext(ctx, "shipping consumer: invalid orderId", "raw", raw, "error", err)
		return nil
	}
	_, err = c.creator.CreateShipmentForOrder(ctx, orderID)
	if errors.Is(err, dberr.ErrConflict) {
		return nil
	}
	return err
}
