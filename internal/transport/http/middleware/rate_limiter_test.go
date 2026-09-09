package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"go.uber.org/mock/gomock"

	cachemocks "github.com/labib0x9/ffgif/internal/port/cache/mocks"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func newLimiter(t *testing.T, rate, capacity int) (*cachemocks.MockRateLimiter, http.Handler, *int32) {
	t.Helper()
	ctrl := gomock.NewController(t)
	client := cachemocks.NewMockRateLimiter(ctrl)
	limiter := middleware.NewRateLimiter(client, rate, capacity)

	var reached int32
	handler := limiter.Limit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reached, 1)
		w.WriteHeader(http.StatusOK)
	}))
	return client, handler, &reached
}

func callFrom(handler http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// The bucket must be keyed by the client IP alone, and the configured capacity
// and rate must reach the script unchanged and in the right order.
func TestRateLimiter_PassesKeyCapacityAndRateToTheScript(t *testing.T) {
	client, handler, reached := newLimiter(t, 5, 10)

	client.EXPECT().
		RunScript(gomock.Any(), gomock.Eq("rate_limit:ip:192.168.1.100"),
			gomock.Eq(10), gomock.Eq(5), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, capacity, rate int, now int64) (any, error) {
			if now <= 0 {
				t.Errorf("now = %d, want a millisecond timestamp", now)
			}
			return []any{int64(1), int64(0), int64(9)}, nil
		}).
		Times(1)

	rec := callFrom(handler, "192.168.1.100:12345")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if atomic.LoadInt32(reached) != 1 {
		t.Error("the request did not reach the handler")
	}
	if got := rec.Header().Get("X-RateLimit-Remaining"); got != "9" {
		t.Errorf("X-RateLimit-Remaining = %q, want 9", got)
	}
}

// The port and any IPv6 brackets must not leak into the bucket key, or every
// connection from one client gets its own bucket and the limiter does nothing.
func TestRateLimiter_KeyIsTheIpWithoutThePort(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		wantKey    string
	}{
		{"ipv4", "203.0.113.7:44321", "rate_limit:ip:203.0.113.7"},
		{"ipv4 different port, same bucket", "203.0.113.7:9999", "rate_limit:ip:203.0.113.7"},
		{"ipv6", "[2001:db8::1]:44321", "rate_limit:ip:2001:db8::1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, handler, _ := newLimiter(t, 5, 10)
			client.EXPECT().
				RunScript(gomock.Any(), gomock.Eq(tc.wantKey), gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]any{int64(1), int64(0), int64(4)}, nil).
				Times(1)

			callFrom(handler, tc.remoteAddr)
		})
	}
}

func TestRateLimiter_ThrottledRequestIsBlocked(t *testing.T) {
	client, handler, reached := newLimiter(t, 5, 10)

	client.EXPECT().
		RunScript(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]any{int64(0), int64(2000), int64(0)}, nil).
		Times(1)

	rec := callFrom(handler, "192.168.1.100:12345")

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", rec.Code)
	}
	if atomic.LoadInt32(reached) != 0 {
		t.Error("a throttled request reached the handler")
	}
	if got := rec.Header().Get("Retry-After"); got != "2" {
		t.Errorf("Retry-After = %q, want 2", got)
	}
}

// EXPECTED TO FAIL: internal/transport/http/middleware/rate_limiter.go computes
//
//	retryAfterSecs := res.wait_ms / 1000
//
// with integer division, so any wait under one second is advertised as
// "Retry-After: 0". A well-behaved client retries immediately, is throttled
// again, and spins — the header actively causes the hammering it exists to
// prevent. RFC 9110 delta-seconds must be rounded UP to at least 1.
func TestRateLimiter_SubSecondWaitIsNotAdvertisedAsZero(t *testing.T) {
	waits := []int64{1, 250, 500, 999}

	for _, waitMs := range waits {
		t.Run("wait_ms="+strconv.FormatInt(waitMs, 10), func(t *testing.T) {
			client, handler, _ := newLimiter(t, 5, 10)
			client.EXPECT().
				RunScript(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]any{int64(0), waitMs, int64(0)}, nil).
				Times(1)

			rec := callFrom(handler, "192.168.1.100:12345")

			if rec.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want 429", rec.Code)
			}
			got := rec.Header().Get("Retry-After")
			if got == "0" {
				t.Errorf("Retry-After = 0 for a %dms wait; clients will retry "+
					"immediately and spin against the limiter", waitMs)
			}
			n, err := strconv.Atoi(got)
			if err != nil {
				t.Fatalf("Retry-After = %q, not an integer: %v", got, err)
			}
			if n < 1 {
				t.Errorf("Retry-After = %d, want >= 1", n)
			}
		})
	}
}

// A limiter that cannot reach Redis must not become an open door; the current
// 500 is a defensible fail-closed choice, so pin it down.
func TestRateLimiter_BackendErrorFailsClosed(t *testing.T) {
	client, handler, reached := newLimiter(t, 5, 10)

	client.EXPECT().
		RunScript(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("redis: connection refused")).
		Times(1)

	rec := callFrom(handler, "192.168.1.100:12345")

	if atomic.LoadInt32(reached) != 0 {
		t.Error("the request was let through while the limiter was blind")
	}
	if rec.Code == http.StatusOK {
		t.Errorf("status = %d, want a non-200 when the limiter backend is down", rec.Code)
	}
}

// EXPECTED TO FAIL (panic): setLimit does an unchecked
// `res.([]interface{})` plus three unchecked `.(int64)` assertions. Any script
// change, a Lua error object, or a Redis client returning []any{...} with a
// different element type takes the whole server down with a panic inside the
// middleware chain rather than returning 500.
func TestRateLimiter_MalformedScriptReplyDoesNotPanic(t *testing.T) {
	replies := []struct {
		name  string
		value any
	}{
		{"not a slice", "OK"},
		{"too few elements", []any{int64(1)}},
		{"float instead of int", []any{"1", "0", "4"}},
		{"nil", nil},
	}

	for _, tc := range replies {
		t.Run(tc.name, func(t *testing.T) {
			client, handler, _ := newLimiter(t, 5, 10)
			client.EXPECT().
				RunScript(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.value, nil).
				Times(1)

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("the middleware panicked on a %s script reply: %v", tc.name, r)
				}
			}()

			rec := callFrom(handler, "192.168.1.100:12345")
			if rec.Code == http.StatusOK {
				t.Errorf("status = %d; a malformed limiter reply must not allow the request", rec.Code)
			}
		})
	}
}

// A RemoteAddr with no port (some proxies, some test setups) must not take the
// request down; the limiter currently answers 500, which is fail-closed.
func TestRateLimiter_RemoteAddrWithoutPort(t *testing.T) {
	_, handler, reached := newLimiter(t, 5, 10)
	// RunScript must not be reached.
	rec := callFrom(handler, "203.0.113.7")

	if atomic.LoadInt32(reached) != 0 {
		t.Error("the request was let through without a rate-limit decision")
	}
	if rec.Code == http.StatusOK {
		t.Errorf("status = %d, want a non-200", rec.Code)
	}
}
