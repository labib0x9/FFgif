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
			return media.LastUploadResponse{}, media.ErrLastVideoNotFound
		}
		return media.LastUploadResponse{}, err
	}
	return videoMetadata, nil
}
