package media

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) GetByKey(ctx context.Context, key string) (media.GifResp, error) {
	return s.gifRepo.GetByKey(ctx, key)
}
