// Package otelx は 5 サービス + shipping-worker が共有する OpenTelemetry の bootstrap を
// 1 箇所に集約する。TracerProvider 初期化と shutdown (flush) をここだけに置き、
// 各 main は Setup を呼んで defer shutdown するだけにする (ADR-[[202606241356]])。
package otelx

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// Setup は TracerProvider と propagator を global に設定し、相関付き slog ハンドラ (SetupLogging) を
// 据えて、flush 用の shutdown を返す。返り値は `defer shutdown()` で呼ぶ前提。
//
// exporter 接続失敗を致命にしない (ADR-[[202606241356]] graceful degradation): OTLP の宛先 env が
// 無ければ exporter を付けず、TracerProvider だけ立てる。AlwaysSample なので exporter が無くても
// SpanContext は有効で traceparent 伝播は効く。Alloy 不在のまま endpoint だけ設定された場合も、
// otlptracegrpc は遅延接続でここでは error にならず、export は background で retry される。
func Setup(ctx context.Context, serviceName string) (func(), error) {
	// exporter 構築前に据え、以降の error 経路でも相関付き JSON ログが出るようにする。
	SetupLogging()

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// resource.New は後勝ちでマージするので、OTEL_SERVICE_NAME が立っていれば serviceName 既定を
	// 上書きさせる。
	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, err
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}
	if otlpEndpointConfigured() {
		exp, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)

	return func() {
		// 終了シグナル後に呼ぶため、応答しない exporter で停止を引きずらせない。
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			slog.Warn("otel tracer provider shutdown failed", "error", err)
		}
	}, nil
}

// 宛先 env が未指定なら (テスト / e2e / CI のように) Alloy が居ない前提とみなし exporter を組まない。
func otlpEndpointConfigured() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
}
