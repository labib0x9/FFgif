package user

import (
	"github.com/google/uuid"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/auth"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/user"
)

func (s *service) DeleteUser(id string, pass string) error {
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

	if err := s.authRepo.DeleteById(uuid); err != nil {
		// http.Error(w, "invalid credentials", http.StatusUnauthorized)
		// slog.Warn("Login: password mismatched", "error", err, "user_id", id)
		return user.ErrTableUpdateFailed
	}
	return nil
}
