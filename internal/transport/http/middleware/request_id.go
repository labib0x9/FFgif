package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func RequestId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := httputil.WithLoggerContext(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
