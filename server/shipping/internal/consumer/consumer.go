// Package consumer は決済確定イベント (Redis Streams) を購読して配送 (shipment) を
// 手配する。配送のトリガは決済確定であって注文確定ではない (ADR-[[202606211200]])。
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

	"github.com/rin2yh/study-architecture/server/internal/dberr"
	"github.com/rin2yh/study-architecture/server/shipping/internal/db"
)

const (
	streamPaymentEvents = "payment.events"
	consumerGroup       = "shipping"
	typePaymentSettled  = "payment.settled"
)

type ShipmentCreator interface {
	CreateShipmentForOrder(ctx context.Context, orderID int64) (db.ShippingShipment, error)
}

type Consumer struct {
	rdb     *redis.Client
	creator ShipmentCreator
	name    string
	block   time.Duration
	backoff time.Duration
}

func NewRedisClient() (*redis.Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		return nil, errors.New("REDIS_URL is required")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opt), nil
}

func New(rc *redis.Client, creator ShipmentCreator) *Consumer {
	// 同一グループ内で consumer を識別する名前。再起動後も pending を引き取れるよう
	// ランダムでなく安定値 (hostname) にする。
	name, _ := os.Hostname()
	if name == "" {
		name = consumerGroup
	}
	return &Consumer{rdb: rc, creator: creator, name: name, block: 5 * time.Second, backoff: time.Second}
}

func (c *Consumer) Run(ctx context.Context) error {
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	slog.Info("shipping consumer started", "stream", streamPaymentEvents, "group", consumerGroup, "consumer", c.name)
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
			case <-time.After(c.backoff):
			}
		}
	}
}

func (c *Consumer) ensureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, streamPaymentEvents, consumerGroup, "$").Err()
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
		Streams:  []string{streamPaymentEvents, ">"},
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
			if err := c.handle(ctx, m.Values); err != nil {
				// ack せず pending に残し、次回 (XReadGroup の再配送) に委ねる。
				slog.Error("shipping consumer: handle failed", "id", m.ID, "error", err)
				continue
			}
			if err := c.rdb.XAck(ctx, streamPaymentEvents, consumerGroup, m.ID).Err(); err != nil {
				slog.Warn("shipping consumer: xack failed", "id", m.ID, "error", err)
			}
		}
	}
	return nil
}

func (c *Consumer) handle(ctx context.Context, values map[string]any) error {
	if t, _ := values["event"].(string); t != typePaymentSettled {
		return nil
	}
	raw, _ := values["orderId"].(string)
	orderID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		// 壊れた payload は再配送しても直らない。pending を膨らませないため握って可視化のみ。
		slog.Error("shipping consumer: invalid orderId", "raw", raw, "error", err)
		return nil
	}
	_, err = c.creator.CreateShipmentForOrder(ctx, orderID)
	if errors.Is(err, dberr.ErrConflict) {
		return nil
	}
	return err
}
