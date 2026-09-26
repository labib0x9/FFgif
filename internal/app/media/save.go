package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Save(ctx context.Context, userId, key string) error {
	owner, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("gifRepo.GetOwner: %w: %w", media.ErrGifNotFound, err)
		}
		return fmt.Errorf("gifRepo.GetOwner: %w", err)
	}

	if owner != userId {
		return media.ErrGifOwnerMismatch
	}

	return s.gifRepo.SaveRecent(ctx, key)
}
