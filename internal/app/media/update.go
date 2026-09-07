package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Update(ctx context.Context, userId string, key string, _gif media.GifUpdateRequest, lastUpdatedAt string) (*media.GifResponse, error) {
	ownerId, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, media.ErrGifNotFound
		}
		return nil, err
	}

	if ownerId != userId {
		return nil, media.ErrGifOwnerMismatch
	}

	resp, err := s.tnx.WithRC(ctx, func(ctx context.Context) (any, error) {
		resp, err := s.gifRepo.GetByKey(ctx, key, true)
		if err != nil {
			return nil, err
		}

		curUpdatedAt := resp.UpdatedAt.Format(time.RFC3339Nano)
		if curUpdatedAt != lastUpdatedAt {
			return nil, media.ErrETagValidationFailed
		}
		return s.gifRepo.Update(ctx, key, _gif)
	})

	if err != nil {
		return nil, err
	}

	updatedGif, ok := resp.(media.GifResponse)
	if !ok {
		return nil, fmt.Errorf("gif type assetion failed")
	}

	return &updatedGif, nil
}
