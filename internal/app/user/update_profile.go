package user

import (
	"context"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

func (s *service) UpdateProfile(ctx context.Context, profile user.ProfileUpdateRequest, userId, lastUpdatedAt string) (*user.ProfileResponse, error) {
	resp, err := s.tnx.WithRC(ctx, func(ctx context.Context) (any, error) {
		resp, err := s.userRepo.GetProfile(ctx, userId, true)
		if err != nil {
			return nil, err
		}

		curUpdatedAt := resp.UpdatedAt.Format(time.RFC3339Nano)
		if curUpdatedAt != lastUpdatedAt {
			return nil, media.ErrETagValidationFailed
		}
		return s.userRepo.UpdateProfile(ctx, profile, userId)
	})

	if err != nil {
		return nil, err
	}

	updatedProfile, ok := resp.(user.ProfileResponse)
	if !ok {
		return nil, fmt.Errorf("gif type assetion failed")
	}

	return &updatedProfile, nil

}
