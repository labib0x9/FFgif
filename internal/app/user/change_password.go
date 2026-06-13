package user

import (
	"github.com/google/uuid"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/auth"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/user"
)

func (s *service) ChangePassword(id string, currentPass string, pass string, confirmPass string) error {
	uuid, err := uuid.Parse(id)

	found, err := s.authRepo.GetById(uuid)
	if err != nil {
		return auth.ErrUserNotFound
	}

	if !s.hasher.CompareHashAndPassword(found.PasswordHash, pass) {
		// http.Error(w, "invalid credentials", http.StatusUnauthorized)
		// slog.Warn("Login: password mismatched", "error", err, "user_id", id)
		return user.ErrPasswordMismatched
	}

	newPassHash, err := s.hasher.GenerateHash(pass)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("ChangePassword: hash generation failed", "error", err)
		return user.ErrHashGenFailed
	}

	err = s.userRepo.ChangePassword(id, newPassHash)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("ChangePassword: user not found", "err", err, "id", id)
		return user.ErrTableUpdateFailed
	}
	return nil
}
