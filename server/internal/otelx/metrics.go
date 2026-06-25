package otelx

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	opts := []sdkmetric.Option{sdkmetric.WithResource(res)}
	if metricsEndpointConfigured() {
		exp, err := otlpmetricgrpc.New(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))
	}
	return sdkmetric.NewMeterProvider(opts...), nil
}

func metricsEndpointConfigured() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT") != ""
}
