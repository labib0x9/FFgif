package share

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/share"
)

// shareBy = id of user, sharedWith = email of user
func (s *service) Create(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error {
	sharedUser, err := s.authRepo.GetByEmail(ctx, sharedWith)
	if err != nil {
		return auth.ErrUserNotFound
	}
	_, err = s.gifRepo.GetByKey(ctx, gifKey)
	if err != nil {
		return media.ErrGifNotFound
	}

	share := share.Share{
		GifKey:     gifKey,
		OwnerID:    sharedBy,
		SharedWith: sharedUser.Id.String(),
		ExpiresAt:  &expiresAt,
	}

	return s.shareRepo.Create(ctx, share)
}
