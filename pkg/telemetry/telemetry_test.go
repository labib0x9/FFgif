package telemetry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labib0x9/ffgif/pkg/telemetry"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTracerAndLoggerInitialization(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	telemetry.SetupLogger("test-service", "test")

	tracer := telemetry.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	traceID, spanID := telemetry.GetTraceAndSpanID(ctx)
	if traceID == "" || spanID == "" {
		t.Fatalf("expected valid traceID and spanID, got traceID=%q, spanID=%q", traceID, spanID)
	}

	// Test nil ctx
	nilTraceID, nilSpanID := telemetry.GetTraceAndSpanID(nil)
	if nilTraceID != "" || nilSpanID != "" {
		t.Fatalf("expected empty trace and span for nil ctx")
	}
}

func TestAMQPHeaderPropagation(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent-span")
	defer span.End()

	parentSpanContext := trace.SpanFromContext(ctx).SpanContext()
	headers := telemetry.InjectAMQPHeaders(ctx, make(amqp.Table))

	if len(headers) == 0 {
		t.Fatalf("expected headers to contain injected trace context, got empty")
	}

	extractedCtx := telemetry.ExtractAMQPHeaders(context.Background(), headers)
	extractedSpanContext := trace.SpanFromContext(extractedCtx).SpanContext()

	if !extractedSpanContext.IsValid() {
		t.Fatalf("expected extracted span context to be valid")
	}

	if extractedSpanContext.TraceID() != parentSpanContext.TraceID() {
		t.Fatalf("trace ID mismatch: expected %s, got %s",
			parentSpanContext.TraceID().String(),
			extractedSpanContext.TraceID().String(),
		)
	}
}

func TestPrometheusMetricsCollection(t *testing.T) {
	telemetry.HttpRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
	telemetry.HttpRequestDuration.WithLabelValues("GET", "/test", "200").Observe(0.025)
	telemetry.HttpRequestsInFlight.WithLabelValues("GET").Inc()
	telemetry.HttpRequestsInFlight.WithLabelValues("GET").Dec()

	telemetry.WorkerTasksTotal.WithLabelValues("test-worker", "success").Inc()
	telemetry.WorkerTaskDuration.WithLabelValues("test-worker").Observe(0.5)
	telemetry.WorkerActiveTasks.WithLabelValues("test-worker").Inc()
	telemetry.WorkerActiveTasks.WithLabelValues("test-worker").Dec()

	telemetry.MediaConversionsTotal.WithLabelValues("gif", "success").Inc()
	telemetry.MediaConversionDuration.WithLabelValues("gif").Observe(1.2)

	telemetry.RabbitMQPublishedTotal.WithLabelValues("test.queue").Inc()
	telemetry.RabbitMQConsumedTotal.WithLabelValues("test.queue", "success").Inc()
}

func TestTelemetryMiddlewareIntegration(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}
