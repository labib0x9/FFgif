package media

import (
	"context"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Stream(ctx context.Context, userId string, key string) (*media.StreamResult, error) {
	expiry := 1 * time.Minute
	url, err := s.storage.GetStreamURL(ctx, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("storage.GetStreamURL: %w", err)
	}
	return &media.StreamResult{
		PresignedUrl: url.String(),
		ExpireIn:     int(expiry.Seconds()),
	}, nil
}
