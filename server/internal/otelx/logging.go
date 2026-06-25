package otelx

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// SetupLogging は stdout への JSON 出力を slog default に据え、ctx の span から trace_id / span_id を
// 付与する相関ハンドラでラップする。これでトレースとログを Grafana 上で相互に辿れる (ADR-[[202606241356]])。
// アプリはログ送信を持たず stdout に出すだけで、収集は Alloy が担う。
func SetupLogging() {
	base := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(traceHandler{base}))
}

// 呼び出し側が slog.*Context を使う前提で、ハンドラ差し替えだけで全ログに相関キーを波及させる。
type traceHandler struct {
	slog.Handler
}

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

// 埋め込みのままだと派生 logger が素の Handler を返し、Handle override が外れてしまうため。
func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{h.Handler.WithGroup(name)}
}
