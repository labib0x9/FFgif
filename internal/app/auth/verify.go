package auth

import (
	"database/sql"
	"errors"

	"github.com/labib0x9/ProjectUnsafe/internal/domain/auth"
	tokenpkg "github.com/labib0x9/ProjectUnsafe/pkg/token"
)

func (s *service) Verify(token string) error {
	hash := tokenpkg.GetTokenHash(token)
	verifier, err := s.verifierRepo.GetByHash(hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// http.Error(w, "token expired or invalid", http.StatusGone)
			// slog.Warn("Verify: token expired", "error", err, "token_hash", hash)
			return auth.ErrInvalidToken
		}
		// http.Error(w, "internal server error, try again later", http.StatusInternalServerError)
		// slog.Error("Verify: token fetch error", "error", err, "token_hash", hash)
		return auth.ErrTokenFetchFailed
	}

	if err := s.authRepo.SetVerified(verifier.UserId); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Verify: user verification error", "error", err, "user_id", verifier.UserId, "token_hash", hash)
		return auth.ErrSetUserVerifiedFailed
	}

	if err := s.verifierRepo.Delete(verifier.Id); err != nil {
		// slog.Error("Verify: failed to delete verifier", "error", err, "id", verifier.Id)
	}
	return nil
}
