package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) GetByKey(ctx context.Context, userId string, key string) (*media.GifResponse, error) {
	ownerId, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("gifRepo.GetOwner: %w: %w", media.ErrGifNotFound, err)
		}
		return nil, fmt.Errorf("gifRepo.GetOwner: %w", err)
	}

	if ownerId != userId {
		return nil, media.ErrGifOwnerMismatch
	}

	gif, err := s.gifRepo.GetByKey(ctx, key, false)
	if err != nil {
		return nil, fmt.Errorf("gifRepo.GetByKey: %w: %w", media.ErrGifFetchFailed, err)
	}
	return &gif, nil
}
