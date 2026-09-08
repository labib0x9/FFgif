package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) LastVideo(ctx context.Context, userId string) (media.LastUploadResponse, error) {
	videoMetadata, err := s.lastVideoRepo.GetLastVideo(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return media.LastUploadResponse{}, fmt.Errorf("lastVideoRepo.GetLastVideo: %w: %w", media.ErrLastVideoNotFound, err)
		}
		return media.LastUploadResponse{}, fmt.Errorf("lastVideoRepo.GetLastVideo: %w", err)
	}
	return videoMetadata, nil
}
