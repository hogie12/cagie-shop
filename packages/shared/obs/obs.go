// Package obs sets up tracing (OpenTelemetry) and structured JSON logs the
// same way in every service.
package obs

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Setup installs the global tracer provider, propagator and JSON logger.
//
// Traces are exported only when OTEL_EXPORTER_OTLP_ENDPOINT is set (e.g.
// http://jaeger:4318). Without it, trace ids are still created, so logs from
// one request still share an id. OTEL_TRACES_SAMPLER_ARG (0..1, default 1)
// keeps that share of new traces.
func Setup(ctx context.Context, service string) (shutdown func(context.Context) error, err error) {
	slog.SetDefault(slog.New(traceHandler{slog.NewJSONHandler(os.Stdout, nil)}).With("service", service))

	ratio := 1.0
	if v, err := strconv.ParseFloat(os.Getenv("OTEL_TRACES_SAMPLER_ARG"), 64); err == nil && v >= 0 && v <= 1 {
		ratio = v
	}
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", service))),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		exp, err := otlptracehttp.New(ctx) // reads the endpoint from the environment
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
		slog.Info("tracing enabled", "endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"), "sample_ratio", ratio)
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp.Shutdown, nil
}

// traceHandler adds trace_id and span_id to every log line written with a
// context (slog.InfoContext etc.), so logs and traces can be matched.
type traceHandler struct{ slog.Handler }

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.Handler.Handle(ctx, r)
}

func (h traceHandler) WithAttrs(a []slog.Attr) slog.Handler {
	return traceHandler{h.Handler.WithAttrs(a)}
}
func (h traceHandler) WithGroup(n string) slog.Handler { return traceHandler{h.Handler.WithGroup(n)} }

func notHealth(r *http.Request) bool { return r.URL.Path != "/healthz" }

// Middleware traces each request, except health checks.
func Middleware(service string) gin.HandlerFunc {
	return otelgin.Middleware(service, otelgin.WithFilter(notHealth))
}

// RequestLog writes one JSON line per request (method, path, status, time),
// carrying the request's trace id. Bodies are never logged.
func RequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if !notHealth(c.Request) {
			return
		}
		attrs := []any{
			"method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(), "ip", c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			slog.ErrorContext(c.Request.Context(), "request failed", append(attrs, "error", c.Errors.String())...)
			return
		}
		slog.InfoContext(c.Request.Context(), "request", attrs...)
	}
}
