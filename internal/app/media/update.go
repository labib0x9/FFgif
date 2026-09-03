package media

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Update(ctx context.Context, userId, key string) error {
	// to-do validate userId
	var req media.GifResponse
	return s.gifRepo.Update(ctx, key, req)
}
