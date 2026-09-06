package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) GetProfile(ctx context.Context, id string) (*user.ProfileResponse, error) {
	profile, err := s.userRepo.GetProfile(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepo.GetProfile: %w", err)
	}
	return &profile, nil
}
