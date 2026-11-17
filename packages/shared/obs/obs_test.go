package obs

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestLogLinesCarryTheTraceID(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(traceHandler{slog.NewJSONHandler(&buf, nil)}).With("service", "test")
	ctx, span := sdktrace.NewTracerProvider().Tracer("t").Start(context.Background(), "op")
	log.InfoContext(ctx, "hello")
	span.End()

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	if line["trace_id"] != span.SpanContext().TraceID().String() || line["service"] != "test" || line["span_id"] == nil {
		t.Fatalf("log line = %v", line)
	}

	buf.Reset()
	log.Info("no context")
	if bytes.Contains(buf.Bytes(), []byte("trace_id")) {
		t.Fatal("a line without a span must not invent a trace id")
	}
}
