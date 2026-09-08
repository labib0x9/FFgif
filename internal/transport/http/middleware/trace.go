package middleware

import (
	"fmt"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Trace returns an HTTP middleware that extracts or creates distributed OpenTelemetry traces,
// populates trace context, records HTTP semantic attributes, and injects X-Trace-ID into response headers.
func Trace(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.GetTracerProvider().Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming request headers
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(
				ctx,
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLPath(r.URL.Path),
					semconv.URLScheme(r.URL.Scheme),
					semconv.UserAgentOriginal(r.UserAgent()),
					attribute.String("client.address", r.RemoteAddr),
				),
			)
			defer span.End()

			// Inject trace ID into response header
			if span.SpanContext().IsValid() {
				traceID := span.SpanContext().TraceID().String()
				w.Header().Set("X-Trace-ID", traceID)
			}

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			r = r.WithContext(ctx)

			next.ServeHTTP(rec, r)

			// Record response status
			span.SetAttributes(semconv.HTTPResponseStatusCode(rec.status))
			if rec.status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d error", rec.status))
			} else {
				span.SetStatus(codes.Ok, "OK")
			}
		})
	}
}

// Ensure telemetry package import is used
var _ = telemetry.GetTraceAndSpanID
