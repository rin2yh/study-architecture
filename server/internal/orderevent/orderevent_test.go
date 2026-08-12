package orderevent_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/orderevent"
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

// 発行側は outbox 行に traceparent を持たせるため Traceparent で取り出す (Inject は持たない)。
func TestTraceparentLinkRoundTrip(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx, want := sampledContext(t)

	values := orderevent.Cancelled{OrderID: 20}.Values()
	values[orderevent.FieldTraceparent] = orderevent.Traceparent(ctx)

	link := orderevent.LinkFrom(context.Background(), values)
	if got := link.SpanContext.TraceID(); got != want {
		t.Fatalf("link trace id = %s, want %s", got, want)
	}
}

// 旧 producer や計装オフでは traceparent が載らないが、その場合も consumer は動き続ける。
func TestLinkFromMissingTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	link := orderevent.LinkFrom(context.Background(), map[string]any{})
	if link.SpanContext.IsValid() {
		t.Fatalf("expected invalid span context for missing traceparent, got valid")
	}
}
