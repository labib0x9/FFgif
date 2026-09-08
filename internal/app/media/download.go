package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Download(ctx context.Context, userId, key string) (string, error) {
	ownerId, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("gifRepo.GetOwner: %w: %w", media.ErrGifNotFound, err)
		}
		return "", fmt.Errorf("gifRepo.GetOwner: %w", err)
	}

	sharedOwnerId, err := s.shareRepo.GetOwner(ctx, userId, key)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("shareRepo.GetOwner: %w", err)
	}

	if ownerId != userId && sharedOwnerId != ownerId {
		return "", media.ErrGifOwnerMismatch
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	url, err := s.storage.Download(ctx, key, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("storage.Download: %w", err)
	}
	return url.String(), nil
}
