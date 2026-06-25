// Package otelx は 5 サービス + shipping-worker が共有する OpenTelemetry の bootstrap を
// 1 箇所に集約する。TracerProvider / MeterProvider の初期化と shutdown (flush) をここだけに置き、
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

// Setup は TracerProvider / MeterProvider と propagator を global に設定し、相関付き slog ハンドラ
// (SetupLogging) を据えて、flush 用の shutdown を返す。返り値は `defer shutdown()` で呼ぶ前提。
//
// MeterProvider を global に置くのは、httpx の otelgin / otelhttp が既定で global の MeterProvider を
// 引くため。これだけでサーバ側 / クライアント側 RED が追加コード無しで計測される。
//
// exporter 接続失敗を致命にしない (ADR-[[202606241356]] graceful degradation): OTLP の宛先 env が
// 無ければ exporter を付けず、provider だけ立てる。AlwaysSample なので exporter が無くても
// SpanContext は有効で traceparent 伝播は効く。Alloy 不在のまま endpoint だけ設定された場合も、
// otlp*grpc は遅延接続でここでは error にならず、export は background で retry される。
func Setup(ctx context.Context, serviceName string) (func(), error) {
	// exporter 構築前に据え、以降の error 経路でも相関付き JSON ログが出るようにする。
	SetupLogging()

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	res, err := newResource(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	tp, err := newTracerProvider(ctx, res)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tp)

	mp, err := newMeterProvider(ctx, res)
	if err != nil {
		return nil, err
	}
	otel.SetMeterProvider(mp)

	return func() {
		// 終了シグナル後に呼ぶため、応答しない exporter で停止を引きずらせない。
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			slog.Warn("otel tracer provider shutdown failed", "error", err)
		}
		if err := mp.Shutdown(shutdownCtx); err != nil {
			slog.Warn("otel meter provider shutdown failed", "error", err)
		}
	}, nil
}

// resource.New は後勝ちでマージするので、OTEL_SERVICE_NAME が立っていれば serviceName 既定を
// 上書きさせる。
func newResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithFromEnv(),
	)
}

func newTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}
	if tracesEndpointConfigured() {
		exp, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}
	return sdktrace.NewTracerProvider(opts...), nil
}

// 宛先 env が未指定なら (テスト / e2e / CI のように) Alloy が居ない前提とみなし exporter を組まない。
func tracesEndpointConfigured() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
}
