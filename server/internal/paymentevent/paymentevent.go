// Package paymentevent は payment→shipping の決済確定イベントの wire 契約を一元的に定める。
// producer (payment) と consumer (shipping) が文字列を各自で持つと無言で配送経路が切れるため、
// トピック名・イベント種別・フィールドキー・ペイロード・trace 伝播をここだけに置く。
package paymentevent

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/eventfield"
	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/strconvx"
)

const (
	Topic       = "payment-events"
	TypeSettled = "payment.settled"
)

const (
	FieldEvent       = "event"
	FieldPaymentID   = "paymentId"
	FieldOrderID     = order.FieldID
	FieldAmountCents = "amountCents"
	// W3C propagator が使うキー。伝播フィールドは traceparent のみで秘匿情報は混ぜない
	// (ADR-[[202606250159]] / ADR-[[202606250141]])。
	FieldTraceparent = "traceparent"
)

type Settled struct {
	PaymentID   int64
	OrderID     order.ID
	AmountCents int64
}

func (s Settled) EventType() string { return TypeSettled }

func (s Settled) AggregateID() string { return strconvx.FormatInt64(s.PaymentID) }

func (s Settled) Values() map[string]any {
	return map[string]any{
		FieldEvent:       TypeSettled,
		FieldPaymentID:   s.PaymentID,
		FieldOrderID:     s.OrderID.String(),
		FieldAmountCents: s.AmountCents,
	}
}

// 同じトピックに他種が流れるため、consumer はこれだけを「素通しして ack」に倒し、壊れた payload
// (DLQ 行き) と区別する。判定を別関数に分けると呼び忘れても気づけないので、復元の結果として返す。
var ErrNotSettled = errors.New("not a payment.settled event")

// consumer に map のキーと型アサーションを手書きさせないための唯一の復元口。欠落・型違いはゼロ値へ
// 化けず error になる (ADR-[[202608160730]])。
func ParseSettled(values map[string]any) (Settled, error) {
	// 種別を名乗らない payload は「別種」ではなく壊れているため。
	t, err := eventfield.Required[string](values, FieldEvent)
	if err != nil {
		return Settled{}, err
	}
	if t != TypeSettled {
		return Settled{}, fmt.Errorf("%w: %s = %q", ErrNotSettled, FieldEvent, t)
	}
	paymentID, err := eventfield.Required[int64](values, FieldPaymentID)
	if err != nil {
		return Settled{}, err
	}
	orderID, err := order.ParseIDFromEvent(values)
	if err != nil {
		return Settled{}, fmt.Errorf("%s: %w", FieldOrderID, err)
	}
	amountCents, err := eventfield.Required[int64](values, FieldAmountCents)
	if err != nil {
		return Settled{}, err
	}
	return Settled{PaymentID: paymentID, OrderID: orderID, AmountCents: amountCents}, nil
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
