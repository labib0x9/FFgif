package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	tokenpkg "github.com/labib0x9/ffgif/pkg/token"
)

func (s *service) Verify(ctx context.Context, token string) error {
	hash := tokenpkg.GetTokenHash(token)
	verifier, err := s.verifierRepo.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrInvalidToken
		}
		return auth.ErrTokenFetchFailed
	}

	_, err = s.tnx.With(ctx, func(ctx context.Context) (any, error) {
		if err := s.authRepo.SetVerified(ctx, verifier.UserId); err != nil {
			return nil, auth.ErrSetUserVerifiedFailed
		}

		return nil, s.verifierRepo.Delete(ctx, verifier.Id)
	})
	return err
}
