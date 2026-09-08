package media

import (
	"context"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

type GifResult struct {
	Data  []media.GifResponse `json:"data"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

func (s *service) GetGifs(ctx context.Context, userId string, filter string) (*GifResult, error) {
	gifs, err := s.gifRepo.Get(ctx, userId, filter)
	if err != nil {
		return nil, fmt.Errorf("gifRepo.Get: %w: %w", media.ErrGifFetchFailed, err)
	}

	return &GifResult{
		Data:  gifs,
		Page:  1,
		Limit: 20,
		Total: len(gifs),
	}, nil
}
