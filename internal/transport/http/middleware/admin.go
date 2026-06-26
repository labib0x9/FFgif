package middleware

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jwt"
)

func (m *Middlewares) Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, ok := r.Context().Value(claimKey).(jwt.Payload)
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
