package media

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Download(ctx context.Context, userId, key string) (string, error) {
	ownerId, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", media.ErrGifNotFound
		}
		return "", err
	}

	sharedOwnerId, err := s.shareRepo.GetOwner(ctx, userId, key)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	if ownerId != userId && sharedOwnerId != ownerId {
		return "", media.ErrGifOwnerMismatch
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	url, err := s.storage.Download(ctx, key, 5*time.Minute)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
