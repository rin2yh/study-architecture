package otelx

import (
	"context"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestMetricsEndpointConfigured(t *testing.T) {
	type args struct{ general, metrics string }
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"準正常系 宛先未指定なら false", args{"", ""}, false},
		{"正常系 汎用 endpoint で true", args{"http://alloy:4317", ""}, true},
		{"正常系 metrics 個別 endpoint で true", args{"", "http://alloy:4317"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tt.args.general)
			t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", tt.args.metrics)
			if got := metricsEndpointConfigured(); got != tt.want {
				t.Errorf("metricsEndpointConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

// otelgin / otelhttp が global から引く前提なので、いずれの経路でも nil を返さないことが要件。
func buildMeterProvider(t *testing.T) *sdkmetric.MeterProvider {
	t.Helper()
	ctx := context.Background()
	res, err := newResource(ctx, "test")
	if err != nil {
		t.Fatalf("newResource: %v", err)
	}
	mp, err := newMeterProvider(ctx, res)
	if err != nil {
		t.Fatalf("newMeterProvider: %v", err)
	}
	if mp == nil {
		t.Fatal("meter provider is nil")
	}
	return mp
}

func TestNewMeterProvider(t *testing.T) {
	// 宛先未指定 (graceful degradation): exporter を組まず flush 対象が無いので shutdown は必ず成功する。
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")

	mp := buildMeterProvider(t)
	if err := mp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown without exporter should not fail: %v", err)
	}
}

func TestNewMeterProviderWithExporter(t *testing.T) {
	// 宛先ありで OTLP exporter を組む経路。到達不能な exporter の flush で停止を引きずらせない短い期限で止める。
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")

	mp := buildMeterProvider(t)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	// Alloy 不在では flush が期限切れしうるが、ここでは provider が立つことだけを確かめる。
	if err := mp.Shutdown(shutdownCtx); err != nil {
		t.Logf("shutdown with unreachable exporter returned (expected): %v", err)
	}
}
