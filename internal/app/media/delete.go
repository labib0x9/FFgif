package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Delete(ctx context.Context, userId string, key string) error {
	ownerId, err := s.gifRepo.GetOwner(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return media.ErrGifNotFound
		}
		return err
	}
	if ownerId != userId {
		return media.ErrGifOwnerMismatch
	}
	if err := s.gifRepo.Delete(ctx, key); err != nil {
		return fmt.Errorf("gifRepo.Delete: %w: %w", media.ErrDeleteByKeyFailed, err)
	}
	return nil
}
