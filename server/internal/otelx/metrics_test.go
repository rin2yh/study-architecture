package otelx

import (
	"context"
	"testing"
	"time"
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
	// otelgin / otelhttp が global から引く前提なので、いずれの経路でも nil を返さないことが要件。
	type args struct{ endpoint string }
	tests := []struct {
		name string
		args args
	}{
		{"正常系 宛先ありで exporter 付き provider が立つ", args{"localhost:4317"}},
		{"準正常系 宛先未指定でも provider は立つ (graceful degradation)", args{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tt.args.endpoint)
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

			// 宛先ありは到達不能な exporter を抱えるため、flush で停止を引きずらせない短い期限で止める。
			shutdownCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			defer cancel()
			err = mp.Shutdown(shutdownCtx)
			// exporter 無しの経路は flush 対象が無く必ず成功するので、ここだけは error を許さない。
			if tt.args.endpoint == "" && err != nil {
				t.Fatalf("shutdown without exporter should not fail: %v", err)
			}
		})
	}
}
