package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("LOG Middleware", "Addr", r.RemoteAddr, "Method", r.Method, "Path", r.URL.Path, "Took", time.Since(start).String())
	})
}
