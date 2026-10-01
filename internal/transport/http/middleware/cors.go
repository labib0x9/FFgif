package middleware

import (
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if httputil.CheckOrigin(r) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, content-type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE, PATCH")
		// w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
