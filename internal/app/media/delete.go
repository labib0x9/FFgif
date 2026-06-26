package media

import "github.com/labib0x9/ffgif/internal/domain/media"

func (s *service) Delete(key string) error {
	if err := s.gifRepo.Delete(key); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("DeleteGif: Delete() failed", "error", err, "key", key)
		return media.ErrDeleteByKeyFailed
	}
	return nil
}
