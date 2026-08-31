package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

type mockRateLimiterClient struct {
	runScriptFunc func(ctx context.Context, key string, capacity int, rate int, now int64) (interface{}, error)
}

func (m *mockRateLimiterClient) RunScript(ctx context.Context, key string, capacity int, rate int, now int64) (interface{}, error) {
	if m.runScriptFunc != nil {
		return m.runScriptFunc(ctx, key, capacity, rate, now)
	}
	// Return allowed: 1, wait_ms: 0, token: 4
	return []interface{}{int64(1), int64(0), int64(4)}, nil
}

func TestRateLimiter_Allowed(t *testing.T) {
	client := &mockRateLimiterClient{
		runScriptFunc: func(ctx context.Context, key string, capacity int, rate int, now int64) (interface{}, error) {
			return []interface{}{int64(1), int64(0), int64(9)}, nil
		},
	}

	limiter := middleware.NewRateLimiter(client, 5, 10)
	limitMw := limiter.Limit()

	handler := limitMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rec.Code)
	}
	if rec.Header().Get("X-RateLimit-Limit") != "5" {
		t.Errorf("expected X-RateLimit-Limit 5, got %s", rec.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimiter_Throttled(t *testing.T) {
	client := &mockRateLimiterClient{
		runScriptFunc: func(ctx context.Context, key string, capacity int, rate int, now int64) (interface{}, error) {
			// Not allowed: 0, wait_ms: 2000, token: 0
			return []interface{}{int64(0), int64(2000), int64(0)}, nil
		},
	}

	limiter := middleware.NewRateLimiter(client, 5, 10)
	limitMw := limiter.Limit()

	handler := limitMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429 Too Many Requests, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "2" {
		t.Errorf("expected Retry-After 2, got %s", rec.Header().Get("Retry-After"))
	}
}
