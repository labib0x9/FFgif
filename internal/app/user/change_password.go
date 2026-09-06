package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/pkg/apperr"
)

func (s *service) ChangePassword(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	found, err := s.authRepo.GetById(ctx, uuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrUserNotFound
		}
		return err
	}

	if pass != confirmPass {
		return apperr.ErrPasswordMismatched
	}

	if !s.hasher.CompareHashAndPassword(found.PasswordHash, currentPass) {
		return auth.ErrInvalidCredential
	}

	newPassHash, err := s.hasher.GenerateHash(pass)
	if err != nil {
		return apperr.ErrHashGenFailed
	}

	err = s.userRepo.ChangePassword(ctx, id, newPassHash)
	if err != nil {
		return apperr.ErrTableUpdateFailed
	}
	return nil
}
