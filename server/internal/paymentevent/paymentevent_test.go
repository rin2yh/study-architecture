package paymentevent_test

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
	"github.com/rin2yh/study-architecture/server/internal/test/orderid"
)

func sampledContext(t *testing.T) (context.Context, trace.TraceID) {
	t.Helper()
	tid, err := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("trace id: %v", err)
	}
	sid, err := trace.SpanIDFromHex("0123456789abcdef")
	if err != nil {
		t.Fatalf("span id: %v", err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	return trace.ContextWithSpanContext(context.Background(), sc), tid
}

func TestInjectLinkRoundTrip(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx, want := sampledContext(t)

	id, err := order.Parse("2")
	if err != nil {
		t.Fatalf("order.New: %v", err)
	}
	values := paymentevent.Settled{PaymentID: 1, OrderID: id, AmountCents: 300}.Values()
	paymentevent.Inject(ctx, values)

	if _, ok := values[paymentevent.FieldTraceparent].(string); !ok {
		t.Fatalf("traceparent was not injected into values: %#v", values)
	}

	link := paymentevent.LinkFrom(context.Background(), values)
	if got := link.SpanContext.TraceID(); got != want {
		t.Fatalf("link trace id = %s, want %s", got, want)
	}
}

func TestParseSettled(t *testing.T) {
	id := orderid.Must(t, "20")
	settled := paymentevent.Settled{PaymentID: 7, OrderID: id, AmountCents: 300}

	// consumer は notSettled だけを「素通しして ack」に倒すため、どちらの error になるかまで固定する。
	type want struct {
		settled    paymentevent.Settled
		err        bool
		notSettled bool
	}
	tests := []struct {
		name   string
		values map[string]any
		want   want
	}{
		{"正常系 発行した payload をそのまま復元する", settled.Values(), want{settled, false, false}},
		{
			"準正常系 別のイベント種別は素通しできる error になる",
			map[string]any{paymentevent.FieldEvent: "payment.failed"},
			want{paymentevent.Settled{}, true, true},
		},
		{
			"準正常系 種別を名乗らない payload は別種扱いにせず隔離へ倒す",
			map[string]any{paymentevent.FieldOrderID: "20"},
			want{paymentevent.Settled{}, true, false},
		},
		{
			"準正常系 フィールドの欠落はゼロ値へ化けず error になる",
			map[string]any{
				paymentevent.FieldEvent:     paymentevent.TypeSettled,
				paymentevent.FieldPaymentID: int64(7),
				paymentevent.FieldOrderID:   "20",
			},
			want{paymentevent.Settled{}, true, false},
		},
		{
			"準正常系 型違いはゼロ値へ化けず error になる",
			map[string]any{
				paymentevent.FieldEvent:       paymentevent.TypeSettled,
				paymentevent.FieldPaymentID:   "7",
				paymentevent.FieldOrderID:     "20",
				paymentevent.FieldAmountCents: int64(300),
			},
			want{paymentevent.Settled{}, true, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := paymentevent.ParseSettled(tt.values)
			if (err != nil) != tt.want.err {
				t.Fatalf("ParseSettled() error = %v, want error %v", err, tt.want.err)
			}
			if gotNotSettled := errors.Is(err, paymentevent.ErrNotSettled); gotNotSettled != tt.want.notSettled {
				t.Fatalf("errors.Is(%v, ErrNotSettled) = %v, want %v", err, gotNotSettled, tt.want.notSettled)
			}
			if got != tt.want.settled {
				t.Fatalf("ParseSettled() = %+v, want %+v", got, tt.want.settled)
			}
		})
	}
}

// 旧 producer や計装オフでは traceparent が載らないが、その場合も consumer は動き続ける。
func TestLinkFromMissingTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	link := paymentevent.LinkFrom(context.Background(), map[string]any{})
	if link.SpanContext.IsValid() {
		t.Fatalf("expected invalid span context for missing traceparent, got valid")
	}
}
