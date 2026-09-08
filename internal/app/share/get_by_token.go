package share

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/share"
)

func (s *service) GetByToken(ctx context.Context, token string) (share.GifTokenResponse, error) {
	return s.shareRepo.GetByToken(ctx, token)
}
