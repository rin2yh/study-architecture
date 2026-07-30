package redisx_test

import (
	"strconv"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/rin2yh/study-architecture/server/internal/redisx"
)

func TestDLQStream(t *testing.T) {
	type args struct{ stream, group string }
	tests := []struct {
		name string
		args args
		want string
	}{
		{"正常系 同じ stream でも group ごとに退避先が分かれる", args{"payment.events", "shipping"}, "dlq:payment.events:shipping"},
		{"正常系 別 group は別ストリーム", args{"payment.events", "inventory"}, "dlq:payment.events:inventory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redisx.DLQStream(tt.args.stream, tt.args.group); got != tt.want {
				t.Fatalf("DLQStream() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeadLetteredMessageKeepsPayload(t *testing.T) {
	mr, rc := newPending(t, settled)
	claimUntilDLQ(t, mr, rc)

	msgs, err := rc.XRange(t.Context(), redisx.DLQStream(testStream, testGroup), "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("dlq messages = %d, want 1", len(msgs))
	}
	got := msgs[0].Values
	for k, want := range settled {
		if got[k] != want {
			t.Errorf("dlq values[%q] = %v, want %v", k, got[k], want)
		}
	}
	if id, _ := got[redisx.FieldDLQSourceID].(string); id == "" {
		t.Errorf("dlq values[%q] = %v, want 元メッセージの ID", redisx.FieldDLQSourceID, got[redisx.FieldDLQSourceID])
	}
	if want := strconv.Itoa(redisx.MaxDeliveries); got[redisx.FieldDLQDeliveries] != want {
		t.Errorf("dlq values[%q] = %v, want %q", redisx.FieldDLQDeliveries, got[redisx.FieldDLQDeliveries], want)
	}
}

// global の MeterProvider は 1 度しか差し替えられないので、収集を見るテストはここに集約する。
func TestObserveDLQDepth(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))

	mr, rc := newPending(t, settled)
	redisx.ObserveDLQDepth(rc, testStream, testGroup)

	// 退避前も系列が生えていないと、アラートが NoData と「滞留 0」を区別できない。
	if got := collectDLQDepth(t, reader); got != 0 {
		t.Fatalf("depth (退避前) = %d, want 0", got)
	}

	claimUntilDLQ(t, mr, rc)

	if got := collectDLQDepth(t, reader); got != 1 {
		t.Fatalf("depth (退避後) = %d, want 1", got)
	}
}

// アラートの PromQL が名前とラベルに依存するので、収集値と一緒にそこも突き合わせる。
func collectDLQDepth(t *testing.T, reader sdkmetric.Reader) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	want := attribute.NewSet(
		attribute.String("messaging.destination.name", redisx.DLQStream(testStream, testGroup)),
		attribute.String("messaging.consumer.group.name", testGroup),
	)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			g, ok := m.Data.(metricdata.Gauge[int64])
			if m.Name != "messaging.dlq.depth" || !ok {
				continue
			}
			for _, dp := range g.DataPoints {
				if dp.Attributes.Equals(&want) {
					return dp.Value
				}
			}
		}
	}
	t.Fatalf("messaging.dlq.depth{%s} が収集されていない", want.Encoded(attribute.DefaultEncoder()))
	return 0
}
