package media

import (
	"database/sql"
	"errors"

	"github.com/labib0x9/ProjectUnsafe/internal/domain/media"
)

func (s *service) LastVideo(userId string) (media.LastUploadResp, error) {
	videoMetadata, err := s.lastVideoRepo.GetLastVideo(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// http.Error(w, "Not found", http.StatusNotFound)
			// slog.Warn("LastVideo: video metadata found", "err", err)
			return media.LastUploadResp{}, media.ErrLastVideoNotFound
		}
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("LastVideo: video metadata found", "err", err)
		return media.LastUploadResp{}, err
	}
	return videoMetadata, nil
}
