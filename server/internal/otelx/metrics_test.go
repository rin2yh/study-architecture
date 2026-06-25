package otelx

import (
	"context"
	"testing"
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

func TestNewMeterProvider(t *testing.T) {
	// 宛先未指定 (graceful degradation) でも MeterProvider は立つこと。
	// otelgin / otelhttp が global から引く前提なので nil を返さないことが要件。
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "")

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
	if err := mp.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
