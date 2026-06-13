package media

func (s *service) Save(key string) error {
	if err := s.gifRepo.SaveRecent(key); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("SaveRecent: SaveRecent() failed", "error", err, "key", key)
		return err
	}
	return nil
}
