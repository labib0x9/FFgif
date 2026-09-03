package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (m *Middlewares) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: Authorization header missing", "Addr", r.RemoteAddr)
			return
		}

		tokenStr := strings.TrimPrefix(h, "Bearer ")
		data, err := m.jwt.Verify(tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				http.Error(w, "token expired", http.StatusUnauthorized)
				slog.Warn("Auth Middleware: token expired", "Addr", r.RemoteAddr)
				return
			}
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: invalid token", "Addr", r.RemoteAddr)
			return
		}

		key := "token_blocklist:" + tokenStr
		if _, err := m.cache.Get(r.Context(), key); err == nil {
			http.Error(w, "blocklist token", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: token on blocklist", "Addr", r.RemoteAddr)
			return
		}

		ctx := httputil.WithAuthContext(r.Context(), data, tokenStr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
