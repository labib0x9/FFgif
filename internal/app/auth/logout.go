package auth

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/pkg/jwt"
)

func (s *service) Logout(ctx context.Context, jwt string, claims jwt.Payload) error {
	key := "token_blocklist:" + jwt
	expire := time.Until(claims.ExpiresAt.Time)
	if expire <= 0 {
		return nil
	}

	if err := s.cache.Set(ctx, key, "1", expire); err != nil {
		return err
	}
	return nil
}
