package media

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Update(ctx context.Context, key string) error {
	var req media.GifResponse
	return s.gifRepo.Update(ctx, key, req)
}
