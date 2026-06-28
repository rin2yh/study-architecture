package paymentevent_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
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

	values := paymentevent.Settled{PaymentID: 1, OrderID: 2, AmountCents: 300}.Values()
	paymentevent.Inject(ctx, values)

	if _, ok := values[paymentevent.FieldTraceparent].(string); !ok {
		t.Fatalf("traceparent was not injected into values: %#v", values)
	}

	link := paymentevent.LinkFrom(context.Background(), values)
	if got := link.SpanContext.TraceID(); got != want {
		t.Fatalf("link trace id = %s, want %s", got, want)
	}
}

func TestDestinationRoundTrip(t *testing.T) {
	type args struct {
		values map[string]any
	}
	type want struct {
		dest paymentevent.Destination
	}
	full := paymentevent.Destination{Recipient: "山田太郎", PostalCode: "1500001", Prefecture: "東京都", City: "渋谷区", Line1: "神宮前1-2-3"}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"正常系 Values で載せた宛先を DestinationFrom で復元する",
			args{paymentevent.Settled{PaymentID: 1, OrderID: 2, AmountCents: 300, Destination: full}.Values()},
			want{full},
		},
		{
			"準正常系 宛先キーが無い古いイベントは空の宛先に倒す",
			args{map[string]any{paymentevent.FieldEvent: paymentevent.TypeSettled, paymentevent.FieldOrderID: "2"}},
			want{paymentevent.Destination{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := paymentevent.DestinationFrom(tt.args.values); got != tt.want.dest {
				t.Fatalf("DestinationFrom = %#v, want %#v", got, tt.want.dest)
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
