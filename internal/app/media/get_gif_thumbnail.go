package media

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

// // key is the gif key
func (s *service) GetGifThumbnail(ctx context.Context, key string) (string, error) {
	gif, err := s.gifRepo.GetByKey(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", media.ErrGifNotFound
		}
		return "", err
	}

	if gif.ThumbnailUrl == "" {
		return "", media.ErrThumbnailNotFound
	}

	url, err := s.storage.GetThumbnailURL(ctx, gif.ThumbnailUrl)
	if err != nil {
		return "", err
	}

	return url.String(), nil
}
