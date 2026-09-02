package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) DeleteUser(ctx context.Context, id string, pass string) error {
	uuid, err := uuid.Parse(id)

	found, err := s.authRepo.GetById(ctx, uuid)
	if err != nil {
		return auth.ErrUserNotFound
	}

	if !s.hasher.CompareHashAndPassword(found.PasswordHash, pass) {
		return auth.ErrInvalidCredential
	}

	if err := s.authRepo.DeleteById(ctx, uuid); err != nil {
		return user.ErrTableUpdateFailed
	}
	return nil
}
