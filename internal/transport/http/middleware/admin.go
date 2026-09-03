package middleware

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (m *Middlewares) Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, ok := httputil.GetClaims(r.Context())
		if !ok {
			return
		}
		slog.Info("Admin", "middleware", payload.Role)
		if payload.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			slog.Info("Role", r.Header.Get("Role"), "")
			return
		}
		next.ServeHTTP(w, r)
	})
}
