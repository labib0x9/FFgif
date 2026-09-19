package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (m *Middlewares) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqId := httputil.GetRequestID(r.Context())
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_request"`)
			httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "authorization header missing", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: authorization header missing", "request_id", reqId, "Addr", r.RemoteAddr)
			return
		}

		tokenStr := strings.TrimPrefix(h, "Bearer ")
		data, err := m.jwt.Verify(tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="token expired"`)
				httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "invalid token", http.StatusUnauthorized)
				slog.Warn("Auth Middleware: token expired", "request_id", reqId, "Addr", r.RemoteAddr)
				return
			}
			w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="invalid token"`)
			httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "invalid token", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: invalid token", "request_id", reqId, "Addr", r.RemoteAddr)
			return
		}

		key := "token_blocklist:" + tokenStr
		val, err := m.cache.Get(r.Context(), key)
		if err == nil && val != "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="token on blocklist"`)
			httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "token on blocklist", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: token on blocklist", "request_id", reqId, "Addr", r.RemoteAddr)
			return
		} else if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="unable to verify token status"`)
			httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unable to verify token status", http.StatusUnauthorized)
			slog.Error("Auth Middleware: blocklist check failed", "request_id", reqId, "Addr", r.RemoteAddr, "err", err)
			return
		}

		ctx := httputil.WithAuthContext(r.Context(), data, tokenStr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
