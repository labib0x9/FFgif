package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTraceMiddleware(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	traceMiddleware := middleware.Trace("test-service")

	handler := traceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	traceID := rec.Header().Get("X-Trace-ID")
	if traceID == "" {
		t.Fatalf("expected X-Trace-ID header in response, got empty")
	}
}

func TestTraceMiddleware_500Error(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	traceMiddleware := middleware.Trace("test-service")

	handler := traceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if rec.Header().Get("X-Trace-ID") == "" {
		t.Fatalf("expected X-Trace-ID header even on error")
	}
}
