package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// ContextHandler wraps an slog.Handler to inject OpenTelemetry trace and span IDs
// into every log record if available in context.
type ContextHandler struct {
	handler slog.Handler
}

// NewContextHandler returns a new ContextHandler wrapping the provided handler.
func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{handler: h}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		span := trace.SpanFromContext(ctx)
		if span != nil && span.SpanContext().IsValid() {
			sc := span.SpanContext()
			r.AddAttrs(
				slog.String("trace_id", sc.TraceID().String()),
				slog.String("span_id", sc.SpanID().String()),
			)
		}
	}
	return h.handler.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{handler: h.handler.WithGroup(name)}
}

// SetupLogger initializes standard structured JSON logging enriched with
// service information and contextual OpenTelemetry trace IDs for Loki ingestion.
func SetupLogger(serviceName, environment string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, opts).WithAttrs([]slog.Attr{
		slog.String("service", serviceName),
		slog.String("env", environment),
	})

	logger := slog.New(NewContextHandler(jsonHandler))
	slog.SetDefault(logger)
}

// GetTraceAndSpanID extracts string representations of trace_id and span_id from ctx.
func GetTraceAndSpanID(ctx context.Context) (traceID string, spanID string) {
	if ctx == nil {
		return "", ""
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return "", ""
	}
	return sc.TraceID().String(), sc.SpanID().String()
}
