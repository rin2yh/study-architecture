package orderevent_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
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

// 発行側は outbox 行に traceparent を持たせるため Traceparent で取り出す (Inject は持たない)。
func TestTraceparentLinkRoundTrip(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx, want := sampledContext(t)

	id, err := order.Parse("20")
	if err != nil {
		t.Fatalf("order.New: %v", err)
	}
	values := orderevent.Cancelled{OrderID: id}.Values()
	values[orderevent.FieldTraceparent] = orderevent.Traceparent(ctx)

	link := orderevent.LinkFrom(context.Background(), values)
	if got := link.SpanContext.TraceID(); got != want {
		t.Fatalf("link trace id = %s, want %s", got, want)
	}
}

func TestParseCancelled(t *testing.T) {
	id := orderid.Must(t, "20")
	cancelled := orderevent.Cancelled{OrderID: id}

	tests := []struct {
		name    string
		values  map[string]any
		want    orderevent.Cancelled
		wantErr bool
	}{
		{"正常系 発行した payload をそのまま復元する", cancelled.Values(), cancelled, false},
		{
			"準正常系 別のイベント種別は復元しない",
			map[string]any{orderevent.FieldEvent: "order.created", orderevent.FieldOrderID: "20"},
			orderevent.Cancelled{}, true,
		},
		{
			"準正常系 フィールドの欠落はゼロ値へ化けず error になる",
			map[string]any{orderevent.FieldEvent: orderevent.TypeCancelled},
			orderevent.Cancelled{}, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orderevent.ParseCancelled(tt.values)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseCancelled() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseCancelled() = %+v, want %+v", got, tt.want)
			}
		})
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
