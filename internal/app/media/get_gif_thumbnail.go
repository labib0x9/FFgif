package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

// // key is the gif key
func (s *service) GetGifThumbnail(ctx context.Context, userId string, key string) (string, error) {
	ownerId, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("gifRepo.GetOwner: %w: %w", media.ErrGifNotFound, err)
		}
		return "", fmt.Errorf("gifRepo.GetOwner: %w", err)
	}

	if ownerId != userId {
		return "", media.ErrGifOwnerMismatch
	}
	gif, err := s.gifRepo.GetByKey(ctx, key, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("gifRepo.GetByKey: %w: %w", media.ErrGifNotFound, err)
		}
		return "", fmt.Errorf("gifRepo.GetByKey: %w: %w", media.ErrGifFetchFailed, err)
	}

	if gif.ThumbnailUrl == "" {
		return "", media.ErrThumbnailNotFound
	}

	url, err := s.storage.GetThumbnailURL(ctx, gif.ThumbnailUrl)
	if err != nil {
		return "", fmt.Errorf("storage.GetThumbnailURL: %w", err)
	}

	return url.String(), nil
}
