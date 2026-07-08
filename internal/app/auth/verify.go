package auth

import (
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	tokenpkg "github.com/labib0x9/ffgif/pkg/token"
)

func (s *service) Verify(token string) error {
	hash := tokenpkg.GetTokenHash(token)
	verifier, err := s.verifierRepo.GetByHash(hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrInvalidToken
		}
		return auth.ErrTokenFetchFailed
	}

	if err := s.authRepo.SetVerified(verifier.UserId); err != nil {
		return auth.ErrSetUserVerifiedFailed
	}

	return s.verifierRepo.Delete(verifier.Id)
}
