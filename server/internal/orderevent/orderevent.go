// Package orderevent は注文キャンセルの補償イベント (order.cancelled) の wire 契約を一元的に定める。
// producer (order) と consumer (payment / shipping / inventory) が文字列を各自で持つと無言で
// 補償経路が切れるため、stream 名・イベント種別・フィールドキー・ペイロード・trace 伝播を
// ここだけに置く。フォワードの paymentevent と対称 (ADR-[[202606261702]])。
package orderevent

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	Stream        = "order.events"
	TypeCancelled = "order.cancelled"
)

const (
	FieldEvent   = "event"
	FieldOrderID = "orderId"
	// W3C propagator が使うキー。伝播フィールドは traceparent のみで秘匿情報は混ぜない
	// (ADR-[[202606250159]] / ADR-[[202606250141]])。
	FieldTraceparent = "traceparent"
)

type Cancelled struct {
	OrderID int64
}

func (c Cancelled) Values() map[string]any {
	return map[string]any{
		FieldEvent:   TypeCancelled,
		FieldOrderID: c.OrderID,
	}
}

// Traceparent は現在の trace の W3C traceparent を返す。計装オフ等で trace が無ければ空文字。
// outbox は送出を後追いするため、発行時点の traceparent をこれで取り出し送信行に保持する。
func Traceparent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get(FieldTraceparent)
}

// LinkFrom は consumer 側で values の traceparent を span link に変換する。発行と消費を親子でなく
// link でつなぐ理由は ADR-[[202606250159]]。
func LinkFrom(ctx context.Context, values map[string]any) trace.Link {
	tp, _ := values[FieldTraceparent].(string)
	carrier := propagation.MapCarrier{FieldTraceparent: tp}
	linkCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
	return trace.LinkFromContext(linkCtx)
}
