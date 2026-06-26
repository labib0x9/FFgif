package media

import "github.com/labib0x9/ffgif/internal/domain/media"

func (s *service) Download(key string) (string, error) {

	gif, err := s.gifRepo.GetByKey(key)
	if err != nil {
		// http.Error(w, "not found", http.StatusNotFound)
		// slog.Error("DownloadGif: GetByKey() failed", "error", err, "key", key)
		return "", media.ErrGifNotFound
	}

	if gif.Status == "private" {
		//
	}

	url := s.gifRepo.GetUrl(key)
	return url, nil
}
