package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type contextKey struct{}
type authHeaderKey struct{}

var claimKey = contextKey{}
var jwtKey = authHeaderKey{}

func (m *Middlewares) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: Authorization header missing", "Addr", r.RemoteAddr)
			return
		}

		tokenStr := strings.TrimPrefix(h, "Bearer ")
		data, err := utils.VerifyJWT(m.Cnf.JwtSecret, tokenStr)
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
		if _, err := m.cacheRepo.Get(key); err == nil {
			http.Error(w, "blocklist token", http.StatusUnauthorized)
			slog.Warn("Auth Middleware: token on blocklist", "Addr", r.RemoteAddr)
			return
		}

		ctx := context.WithValue(r.Context(), claimKey, data)
		ctx = context.WithValue(ctx, jwtKey, tokenStr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetClaims(r *http.Request) (utils.Payload, bool) {
	claims, ok := r.Context().Value(claimKey).(utils.Payload)
	return claims, ok
}

func GetAuthorizationHeader(r *http.Request) (string, bool) {
	token, ok := r.Context().Value(jwtKey).(string)
	return token, ok
}

func GetUserId(r *http.Request) string {
	claims, ok := GetClaims(r)
	if !ok {
		return ""
	}

	return claims.RegisteredClaims.Subject
}
