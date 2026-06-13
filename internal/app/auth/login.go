package auth

import (
	"github.com/google/uuid"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/auth"
)

type Result struct {
	Token string
	Id    uuid.UUID
}

func (s *service) Login(email string, password string) (*Result, error) {
	found, err := s.authRepo.GetByEmail(email)
	if err != nil {
		// http.Error(w, "invalid credentials", http.StatusUnauthorized)
		// slog.Warn("Login: user fetch error", "error", err, "email", req.Email)
		return nil, auth.ErrInvalidCredential
	}

	if !found.IsVerified {
		// http.Error(w, "not verified", http.StatusForbidden)
		// slog.Warn("Login: not verified", "email", req.Email)
		return nil, auth.ErrUserNotVerified
	}

	if !s.hasher.CompareHashAndPassword(found.PasswordHash, password) {
		// http.Error(w, "invalid credentials", http.StatusUnauthorized)
		// slog.Warn("Login: password mismatched", "error", err, "email", req.Email)
		return nil, auth.ErrInvalidCredential
	}

	token, err := s.jwt.Create(
		found.Fullname,
		found.Id.String(),
		found.Email,
		found.Role,
	)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Login: jwt create error", "error", err, "email", req.Email)
		return nil, err
	}

	return &Result{Token: token, Id: found.Id}, nil
}
