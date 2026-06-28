// Package paymentevent は payment→shipping の決済確定イベントの wire 契約を一元的に定める。
// producer (payment) と consumer (shipping) が文字列を各自で持つと無言で配送経路が切れるため、
// stream 名・イベント種別・フィールドキー・ペイロード・trace 伝播をここだけに置く。
package paymentevent

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	Stream      = "payment.events"
	TypeSettled = "payment.settled"
)

const (
	FieldEvent       = "event"
	FieldPaymentID   = "paymentId"
	FieldOrderID     = "orderId"
	FieldAmountCents = "amountCents"
	// 配送先スナップショット (ADR-[[202606261704]])。payment は中身を解釈せず traceparent と同様に
	// 中継するだけで、shipment が settled 経由で宛先を受け取る (shipping→order の同期依存を作らない)。
	FieldShipRecipient  = "shipRecipient"
	FieldShipPostalCode = "shipPostalCode"
	FieldShipPrefecture = "shipPrefecture"
	FieldShipCity       = "shipCity"
	FieldShipLine1      = "shipLine1"
	// W3C propagator が使うキー。伝播フィールドは traceparent のみで秘匿情報は混ぜない
	// (ADR-[[202606250159]] / ADR-[[202606250141]])。
	FieldTraceparent = "traceparent"
)

// Destination は注文時に確定した配送先のスナップショット (ADR-[[202606261704]])。
type Destination struct {
	Recipient  string
	PostalCode string
	Prefecture string
	City       string
	Line1      string
}

type Settled struct {
	PaymentID   int64
	OrderID     int64
	AmountCents int64
	Destination Destination
}

func (s Settled) Values() map[string]any {
	return map[string]any{
		FieldEvent:          TypeSettled,
		FieldPaymentID:      s.PaymentID,
		FieldOrderID:        s.OrderID,
		FieldAmountCents:    s.AmountCents,
		FieldShipRecipient:  s.Destination.Recipient,
		FieldShipPostalCode: s.Destination.PostalCode,
		FieldShipPrefecture: s.Destination.Prefecture,
		FieldShipCity:       s.Destination.City,
		FieldShipLine1:      s.Destination.Line1,
	}
}

// 欠落・型不一致は空文字に倒し、宛先未指定の古いイベントでも consumer を落とさない。
func DestinationFrom(values map[string]any) Destination {
	str := func(k string) string {
		v, _ := values[k].(string)
		return v
	}
	return Destination{
		Recipient:  str(FieldShipRecipient),
		PostalCode: str(FieldShipPostalCode),
		Prefecture: str(FieldShipPrefecture),
		City:       str(FieldShipCity),
		Line1:      str(FieldShipLine1),
	}
}

// Traceparent は現在の trace の W3C traceparent を返す。計装オフ等で trace が無ければ空文字。
// outbox は送出を後追いするため、発行時点の traceparent をこれで取り出し送信行に保持する。
func Traceparent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get(FieldTraceparent)
}

// Inject は producer 側で現在の trace を values に載せる。これが consumer 側の span link の起点になる。
func Inject(ctx context.Context, values map[string]any) {
	if tp := Traceparent(ctx); tp != "" {
		values[FieldTraceparent] = tp
	}
}

// LinkFrom は consumer 側で values の traceparent を span link に変換する。発行と消費を親子でなく
// link でつなぐ理由は ADR-[[202606250159]]。
func LinkFrom(ctx context.Context, values map[string]any) trace.Link {
	tp, _ := values[FieldTraceparent].(string)
	carrier := propagation.MapCarrier{FieldTraceparent: tp}
	linkCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
	return trace.LinkFromContext(linkCtx)
}
