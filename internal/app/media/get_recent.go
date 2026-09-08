package media

import (
	"context"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) GetRecents(ctx context.Context, userId string) ([]media.GifResponse, error) {
	gifs, err := s.gifRepo.GetRecents(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("gifRepo.GetRecents: %w: %w", media.ErrGifFetchFailed, err)
	}
	return gifs, nil
}
