package media

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
)

func (s *service) SaveMetadata(ctx context.Context, msg queue.SaveVideoMessage) error {
	userID, err := uuid.Parse(msg.UserID)
	if err != nil {
		return media.ErrInvalidUserID
	}

	info, err := s.storage.Status(ctx, msg.Key)
	if err != nil {
		if retyErr := s.queue.PublishRetrySaveVideo(ctx, msg); retyErr != nil {
			return errors.Join(err, retyErr)
		}
		return err
	}

	upload := media.LastUpload{
		UserID:      userID,
		FileKey:     msg.Key,
		Filename:    msg.Filename,
		ContentType: info.ContentType,
		SizeBytes:   &info.Size,
		UploadedAt:  info.UploadedAt,
	}

	err = s.lastVideoRepo.Create(ctx, upload)
	return err
}
