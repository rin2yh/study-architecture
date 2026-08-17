// Package orderevent は注文キャンセルの補償イベント (order.cancelled) の wire 契約を一元的に定める。
// producer (order) と consumer (payment / shipping / inventory) が文字列を各自で持つと無言で
// 補償経路が切れるため、トピック名・イベント種別・フィールドキー・ペイロード・trace 伝播を
// ここだけに置く。フォワードの paymentevent と対称 (ADR-[[202606261702]])。
package orderevent

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/eventfield"
	"github.com/rin2yh/study-architecture/server/internal/order"
)

const (
	Topic         = "order-events"
	TypeCancelled = "order.cancelled"
)

const (
	FieldEvent   = "event"
	FieldOrderID = order.FieldID
	// W3C propagator が使うキー。伝播フィールドは traceparent のみで秘匿情報は混ぜない
	// (ADR-[[202606250159]] / ADR-[[202606250141]])。
	FieldTraceparent = "traceparent"
)

type Cancelled struct {
	OrderID order.ID
}

func (c Cancelled) EventType() string { return TypeCancelled }

func (c Cancelled) AggregateID() string { return c.OrderID.String() }

func (c Cancelled) Values() map[string]any {
	return map[string]any{
		FieldEvent:   TypeCancelled,
		FieldOrderID: c.OrderID.String(),
	}
}

// 同じトピックに他種が流れるため、consumer はこれだけを「素通しして ack」に倒し、壊れた payload
// (DLQ 行き) と区別する。判定を別関数に分けると呼び忘れても気づけないので、復元の結果として返す。
var ErrNotCancelled = errors.New("not an order.cancelled event")

// consumer に map のキーと型アサーションを手書きさせないための唯一の復元口。欠落・型違いはゼロ値へ
// 化けず error になる (ADR-[[202608160730]])。
func ParseCancelled(values map[string]any) (Cancelled, error) {
	// 種別を名乗らない payload は「別種」ではなく壊れているため。
	t, err := eventfield.Required[string](values, FieldEvent)
	if err != nil {
		return Cancelled{}, err
	}
	if t != TypeCancelled {
		return Cancelled{}, fmt.Errorf("%w: %s = %q", ErrNotCancelled, FieldEvent, t)
	}
	orderID, err := order.ParseIDFromEvent(values)
	if err != nil {
		return Cancelled{}, fmt.Errorf("%s: %w", FieldOrderID, err)
	}
	return Cancelled{OrderID: orderID}, nil
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
