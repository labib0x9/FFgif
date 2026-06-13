package middleware

import (
	"net/http"
)

// TO-DO
func (m *Middlewares) OneTimePerEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
