// Package paymentevent は payment→shipping の決済確定イベントの wire 契約を一元的に定める。
// producer (payment) と consumer (shipping) が文字列を各自で持つと無言で配送経路が切れるため。
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
	// W3C propagator が使うキーで改名できない (ADR-[[202606250159]] / ADR-[[202606250141]])。
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

// 同じトピックに他種が流れるため、consumer が他種と壊れた payload を区別できるようにする。
var ErrNotSettled = errors.New("not a payment.settled event")

// (ADR-[[202608160730]])
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

// 計装オフ等で trace が無ければ空文字。outbox は送出を後追いするため、発行時点の値を送信行に保持する。
func Traceparent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get(FieldTraceparent)
}

// consumer 側の span link の起点になる。
func Inject(ctx context.Context, values map[string]any) {
	if tp := Traceparent(ctx); tp != "" {
		values[FieldTraceparent] = tp
	}
}

// (ADR-[[202606250159]])
func LinkFrom(ctx context.Context, values map[string]any) trace.Link {
	tp, _ := values[FieldTraceparent].(string)
	carrier := propagation.MapCarrier{FieldTraceparent: tp}
	linkCtx := otel.GetTextMapPropagator().Extract(ctx, carrier)
	return trace.LinkFromContext(linkCtx)
}
