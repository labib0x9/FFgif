package middleware

import "net/http"

func (m *Middlewares) MaxBody1MB(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024 * 1024)
		next.ServeHTTP(w, r)
	})
}
