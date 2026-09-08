package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/media"
)

// streaming key, status, error
func (s *service) Status(ctx context.Context, userId, key string) (string, string, error) {
	lookupKey := "uploading:" + key
	status, err := s.cache.Get(ctx, lookupKey)
	if err != nil {
		return "", "", err
	}
	if status == "" {
		status = "failed"
	}
	if status != "ok" {
		return "", status, nil
	}

	resp, err := s.lastVideoRepo.GetLastVideo(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", status, fmt.Errorf("lastVideoRepo.GetLastVideo: %w: %w", media.ErrLastVideoNotFound, err)
		}
		return "", status, fmt.Errorf("lastVideoRepo.GetLastVideo: %w", err)
	}

	return resp.FileKey, status, nil
}
