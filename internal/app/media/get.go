package media

import (
	"github.com/labib0x9/ffgif/internal/domain/media"
)

type GifResult struct {
	Data  []media.GifResp `json:"data"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

func (s *service) GetGifs(id string, filter string) (*GifResult, error) {
	gifs, err := s.gifRepo.Get(id, filter)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("GetGifs: Get() failed", "error", err, "user_id", id)
		return nil, media.ErrGifFetchFailed
	}

	return &GifResult{
		Data:  gifs,
		Page:  1,
		Limit: 20,
		Total: len(gifs),
	}, nil
}
