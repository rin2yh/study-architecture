package redisx

import (
	"context"
	"log/slog"
	"maps"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// DB 再起動程度の一過性障害では正常なメッセージを退避させない値に取る (隔離までの猶予は
// ClaimMinIdle 倍で効く: ADR-[[202607301418]])。
const MaxDeliveries = 10

const (
	dlqPrefix = "dlq:"
	// 元メッセージの値をそのまま複製するため、退避時のメタ情報は接頭辞付きのキーに逃がす。
	FieldDLQSourceID   = "dlqSourceId"
	FieldDLQDeliveries = "dlqDeliveries"
)

// メッセージごとに引かないよう保持する。otel の global は遅延差し替えに対応するので、
// MeterProvider 設定前に取得しても問題ない。
var meter = otel.Meter("redisx")

// DLQStream は stream / group に対応する退避先ストリーム名を返す。stream だけでなく group でも
// 分けるのは、同じ stream を複数 group が読むため (ADR-[[202607301418]])。
func DLQStream(stream, group string) string {
	return dlqPrefix + stream + ":" + group
}

func deadLetter(ctx context.Context, rdb *redis.Client, stream, group string, m redis.XMessage, deliveries int64) error {
	values := make(map[string]any, len(m.Values)+2)
	maps.Copy(values, m.Values)
	values[FieldDLQSourceID] = m.ID
	values[FieldDLQDeliveries] = deliveries
	return rdb.XAdd(ctx, &redis.XAddArgs{Stream: DLQStream(stream, group), Values: values}).Err()
}

// 消費者のいない DLQ は自分では減らないので、退避の発生回数でなく滞留量で見る
// (ADR-[[202607301418]])。
// 計装の失敗は呼び出し側では扱えないので、ここでログに残して縮退する (ADR-[[202606261216]])。
func ObserveDLQDepth(rdb *redis.Client, stream, group string) {
	dlq := DLQStream(stream, group)
	if err := registerDLQDepth(rdb, dlq, group); err != nil {
		slog.Warn("redisx: dlq depth gauge unavailable", "dlq", dlq, "error", err)
	}
}

func registerDLQDepth(rdb *redis.Client, dlq, group string) error {
	depth, err := meter.Int64ObservableGauge("messaging.dlq.depth",
		metric.WithDescription("Number of messages sitting in the dead letter queue"))
	if err != nil {
		return err
	}
	attrs := metric.WithAttributes(
		semconv.MessagingDestinationName(dlq),
		semconv.MessagingConsumerGroupName(group),
	)
	// 未作成の stream でも XLEN は 0 を返すため、退避が 1 件も起きていない間も系列は生える。
	_, err = meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		n, err := rdb.XLen(ctx, dlq).Result()
		if err != nil {
			return err
		}
		o.ObserveInt64(depth, n, attrs)
		return nil
	}, depth)
	return err
}
