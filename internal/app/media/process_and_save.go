package media

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/domain/media"
)

func (s *service) ProcessAndSave(ctx context.Context, key string) error {
	if key == "" {
		return media.ErrEmptyKey
	}

	res, err := s.processor.PreProcess(ctx, key)
	if err != nil {
		return fmt.Errorf("processor.PreProcess: %w: %w", media.ErrInvalidFiletype, err)
	}

	userId := seperateUserId(key)
	if userId == "" {
		return fmt.Errorf("Empty user id")
	}

	size, err := strconv.ParseInt(res.Size, 10, 64)
	if err != nil {
		return fmt.Errorf("strconv failed")
	}

	userID, err := uuid.Parse(userId)
	if err != nil {
		return fmt.Errorf("user id uuid convertion failed")
	}

	upload := media.LastUpload{
		UserID:       userID,
		FileKey:      res.VideoKey,
		ContentType:  res.ContentType,
		SizeBytes:    &size,
		ThumbnailURL: &res.ThumbnailKey,
		UploadedAt:   time.Now(),
	}

	return s.lastVideoRepo.Create(ctx, upload)
}

func seperateUserId(key string) string {
	return strings.Split(key, string(":"))[0]
}
