package media

import "context"

func (s *service) Save(ctx context.Context, key string) error {
	if err := s.gifRepo.SaveRecent(ctx, key); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("SaveRecent: SaveRecent() failed", "error", err, "key", key)
		return err
	}
	return nil
}
