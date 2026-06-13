package auth

import (
	"context"
	"time"

	"github.com/labib0x9/ProjectUnsafe/pkg/jwt"
)

func (s *service) Logout(ctx context.Context, jwt string, claims jwt.Payload) error {
	key := "token_blocklist:" + jwt
	expire := time.Until(claims.ExpiresAt.Time)
	if expire <= 0 {
		// utils.SendJson(w, "logout", http.StatusOK)
		return nil
	}

	if err := s.cache.Set(ctx, key, "1", expire); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Warn("Logout: failed to blocklist jwt", "Addr", r.RemoteAddr)
		return err
	}
	return nil
}
