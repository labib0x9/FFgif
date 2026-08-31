package media

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) GetRecents(ctx context.Context, id string) ([]media.GifResponse, error) {
	return s.gifRepo.GetRecents(ctx, id)
}
