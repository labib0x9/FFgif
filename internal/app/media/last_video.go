package media

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) LastVideo(ctx context.Context, userId string) (media.LastUploadResponse, error) {
	videoMetadata, err := s.lastVideoRepo.GetLastVideo(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// http.Error(w, "Not found", http.StatusNotFound)
			// slog.Warn("LastVideo: video metadata found", "err", err)
			return media.LastUploadResponse{}, media.ErrLastVideoNotFound
		}
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("LastVideo: video metadata found", "err", err)
		return media.LastUploadResponse{}, err
	}
	return videoMetadata, nil
}
