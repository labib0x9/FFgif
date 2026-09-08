package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
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
		slog.Info("request",
			"request_id", httputil.GetRequestID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"addr", r.RemoteAddr,
			"took", time.Since(start).String(),
		)
	})
}
