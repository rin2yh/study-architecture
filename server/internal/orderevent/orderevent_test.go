package orderevent_test

import (
	"context"
	"errors"
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

// orderevent は Inject を持たないため Traceparent で取り出す。
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

	type want struct {
		cancelled    orderevent.Cancelled
		err          bool
		notCancelled bool
	}
	tests := []struct {
		name   string
		values map[string]any
		want   want
	}{
		{"正常系 発行した payload をそのまま復元する", cancelled.Values(), want{cancelled, false, false}},
		{
			"準正常系 別のイベント種別は素通しできる error になる",
			map[string]any{orderevent.FieldEvent: "order.created", orderevent.FieldOrderID: "20"},
			want{orderevent.Cancelled{}, true, true},
		},
		{
			"準正常系 種別を名乗らない payload は別種扱いにせず隔離へ倒す",
			map[string]any{orderevent.FieldOrderID: "20"},
			want{orderevent.Cancelled{}, true, false},
		},
		{
			"準正常系 フィールドの欠落はゼロ値へ化けず error になる",
			map[string]any{orderevent.FieldEvent: orderevent.TypeCancelled},
			want{orderevent.Cancelled{}, true, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orderevent.ParseCancelled(tt.values)
			if (err != nil) != tt.want.err {
				t.Fatalf("ParseCancelled() error = %v, want error %v", err, tt.want.err)
			}
			if gotNotCancelled := errors.Is(err, orderevent.ErrNotCancelled); gotNotCancelled != tt.want.notCancelled {
				t.Fatalf("errors.Is(%v, ErrNotCancelled) = %v, want %v", err, gotNotCancelled, tt.want.notCancelled)
			}
			if got != tt.want.cancelled {
				t.Fatalf("ParseCancelled() = %+v, want %+v", got, tt.want.cancelled)
			}
		})
	}
}

// 旧 producer や計装オフでは traceparent が載らない。
func TestLinkFromMissingTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	link := orderevent.LinkFrom(context.Background(), map[string]any{})
	if link.SpanContext.IsValid() {
		t.Fatalf("expected invalid span context for missing traceparent, got valid")
	}
}
