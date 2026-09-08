package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func TestMetricsMiddleware(t *testing.T) {
	handler := middleware.Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("metrics test ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestMetricsMiddleware_MetricsPathExempt(t *testing.T) {
	called := false
	handler := middleware.Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected next handler to be called for /metrics")
	}
}
