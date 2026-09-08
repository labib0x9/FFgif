package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labib0x9/ffgif/pkg/telemetry"
)

// Metrics returns an HTTP middleware that collects Prometheus metrics for incoming requests.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Avoid double instrumentation for the /metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		telemetry.HttpRequestsInFlight.WithLabelValues(r.Method).Inc()
		defer telemetry.HttpRequestsInFlight.WithLabelValues(r.Method).Dec()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(rec.status)

		telemetry.HttpRequestDuration.WithLabelValues(r.Method, r.URL.Path, statusCode).Observe(duration)
		telemetry.HttpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
	})
}
