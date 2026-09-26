package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/cache"
)

// streaming key, status, error
func (s *service) Status(ctx context.Context, userId, key string) (string, string, error) {
	if strings.Contains(key, ":") && !strings.HasPrefix(key, userId+":") {
		return "", "", media.ErrGifOwnerMismatch
	}

	lookupKey := "uploading:" + key
	status, err := s.cache.Get(ctx, lookupKey)
	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			status = "failed"
		} else {
			return "", "", err
		}
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
