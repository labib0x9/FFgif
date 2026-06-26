package media

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Download(key string) (string, error) {
	_, err := s.gifRepo.GetByKey(key)
	if err != nil {
		return "", media.ErrGifNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	url, err := s.storage.Download(ctx, key, 5*time.Minute)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
