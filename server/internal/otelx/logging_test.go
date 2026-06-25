package otelx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

const (
	wantTraceID = "0123456789abcdef0123456789abcdef"
	wantSpanID  = "0123456789abcdef"
)

func sampledContext(t *testing.T) context.Context {
	t.Helper()
	tid, err := trace.TraceIDFromHex(wantTraceID)
	if err != nil {
		t.Fatalf("trace id: %v", err)
	}
	sid, err := trace.SpanIDFromHex(wantSpanID)
	if err != nil {
		t.Fatalf("span id: %v", err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(context.Background(), sc)
}

func emit(t *testing.T, build func(slog.Handler) *slog.Logger, ctx context.Context) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	logger := build(traceHandler{slog.NewJSONHandler(&buf, nil)})
	logger.LogAttrs(ctx, slog.LevelInfo, "msg")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line is not JSON (%q): %v", buf.String(), err)
	}
	return line
}

func TestTraceHandler(t *testing.T) {
	plain := func(h slog.Handler) *slog.Logger { return slog.New(h) }

	t.Run("span ありで trace_id / span_id が乗る", func(t *testing.T) {
		line := emit(t, plain, sampledContext(t))
		if line["trace_id"] != wantTraceID {
			t.Errorf("trace_id = %v, want %s", line["trace_id"], wantTraceID)
		}
		if line["span_id"] != wantSpanID {
			t.Errorf("span_id = %v, want %s", line["span_id"], wantSpanID)
		}
	})

	t.Run("span なしでは相関キーを足さない", func(t *testing.T) {
		line := emit(t, plain, context.Background())
		if _, ok := line["trace_id"]; ok {
			t.Errorf("trace_id should be absent without a span: %v", line)
		}
		if _, ok := line["span_id"]; ok {
			t.Errorf("span_id should be absent without a span: %v", line)
		}
	})

	t.Run("With 経由の派生 logger でも相関が続く", func(t *testing.T) {
		withAttrs := func(h slog.Handler) *slog.Logger { return slog.New(h).With("svc", "x") }
		line := emit(t, withAttrs, sampledContext(t))
		if line["trace_id"] != wantTraceID {
			t.Errorf("trace_id = %v, want %s", line["trace_id"], wantTraceID)
		}
	})

	t.Run("WithGroup 経由の派生 logger でも相関が続く", func(t *testing.T) {
		withGroup := func(h slog.Handler) *slog.Logger { return slog.New(h).WithGroup("g") }
		line := emit(t, withGroup, sampledContext(t))
		grp, ok := line["g"].(map[string]any)
		if !ok {
			t.Fatalf("group g missing: %v", line)
		}
		if grp["trace_id"] != wantTraceID {
			t.Errorf("g.trace_id = %v, want %s", grp["trace_id"], wantTraceID)
		}
	})
}
