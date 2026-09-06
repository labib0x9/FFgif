package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/pkg/apperr"
)

func (s *service) DeleteUser(ctx context.Context, id string, pass string) error {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	found, err := s.authRepo.GetById(ctx, uuid)
	if err != nil {
		return fmt.Errorf("authRepo.GetById: %w: %w", auth.ErrUserNotFound, err)
	}

	if !s.hasher.CompareHashAndPassword(found.PasswordHash, pass) {
		return auth.ErrInvalidCredential
	}

	if err := s.authRepo.DeleteById(ctx, uuid); err != nil {
		return fmt.Errorf("authRepo.DeleteById: %w: %w", apperr.ErrTableUpdateFailed, err)
	}
	return nil
}
