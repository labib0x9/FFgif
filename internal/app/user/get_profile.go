package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) GetProfile(ctx context.Context, id string) (*user.ProfileResponse, error) {
	profile, err := s.userRepo.GetProfile(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, auth.ErrUserNotFound
	}
	return &profile, nil
}
