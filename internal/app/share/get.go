package share

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/share"
)

func (s *service) Get(ctx context.Context, user string) ([]share.GifResponse, error) {
	return s.shareRepo.Get(ctx, user)
}
