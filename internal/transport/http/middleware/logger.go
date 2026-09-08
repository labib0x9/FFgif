package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/pkg/telemetry"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)

		traceID, spanID := telemetry.GetTraceAndSpanID(r.Context())
		slog.InfoContext(r.Context(), "request",
			"request_id", httputil.GetRequestID(r.Context()),
			"trace_id", traceID,
			"span_id", spanID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"addr", r.RemoteAddr,
			"took", time.Since(start).String(),
		)
	})
}
