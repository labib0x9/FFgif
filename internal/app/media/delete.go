package media

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) Delete(ctx context.Context, key string) error {
	if err := s.gifRepo.Delete(ctx, key); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("DeleteGif: Delete() failed", "error", err, "key", key)
		return media.ErrDeleteByKeyFailed
	}
	return nil
}
