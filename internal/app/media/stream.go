package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Stream(ctx context.Context, userId string, key string) (*media.StreamResult, error) {
	ownerId, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("gifRepo.GetOwner: %w: %w", media.ErrGifNotFound, err)
		}
		return nil, fmt.Errorf("gifRepo.GetOwner: %w", err)
	}

	if ownerId != userId {
		return nil, media.ErrGifOwnerMismatch
	}
	expiry := 1 * time.Minute
	url, err := s.storage.GetStreamURL(ctx, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("storage.GetStreamURL: %w", err)
	}
	return &media.StreamResult{
		PresignedUrl: url.String(),
		ExpireIn:     int(expiry.Seconds()),
	}, nil
}
