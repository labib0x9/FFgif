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

func (s *service) GetGifs(ctx context.Context, userId string, filter string, page int, limit int) (*GifResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	gifs, err := s.gifRepo.Get(ctx, userId, filter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("gifRepo.Get: %w: %w", media.ErrGifFetchFailed, err)
	}

	return &GifResult{
		Data:  gifs,
		Page:  page,
		Limit: limit,
		Total: len(gifs),
	}, nil
}
