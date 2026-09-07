package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/share"
)

// shareBy = id of user, sharedWith = email of user
func (s *service) Create(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error {
	sharedUser, err := s.authRepo.GetByEmail(ctx, sharedWith)
	if err != nil {
		return fmt.Errorf("authRepo.GetByEmail: %w: %w", auth.ErrUserNotFound, err)
	}

	owner, err := s.gifRepo.GetOwner(ctx, gifKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("gifRepo.GetOwner: %w: %w", media.ErrGifNotFound, err)
		}
		return fmt.Errorf("gifRepo.GetOwner: %w", err)
	}

	if owner != sharedBy {
		return media.ErrGifOwnerMismatch
	}

	share := share.Share{
		GifKey:     gifKey,
		OwnerID:    sharedBy,
		SharedWith: sharedUser.Id.String(),
		ExpiresAt:  &expiresAt,
	}

	return s.shareRepo.Create(ctx, share)
}
