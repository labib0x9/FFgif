package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/domain/auth"
)

type Result struct {
	Token string
	Id    uuid.UUID
}

func (s *service) Login(ctx context.Context, email string, password string) (*Result, error) {
	found, err := s.authRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, auth.ErrInvalidCredential
	}

	if !found.IsVerified {
		return nil, auth.ErrUserNotVerified
	}

	if !s.hasher.CompareHashAndPassword(found.PasswordHash, password) {
		return nil, auth.ErrInvalidCredential
	}

	token, err := s.jwt.Create(
		found.Fullname,
		found.Id.String(),
		found.Email,
		found.Role,
	)
	if err != nil {
		return nil, err
	}

	return &Result{Token: token, Id: found.Id}, nil
}
