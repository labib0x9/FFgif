package job

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/queue"
)

func (s *service) Process(ctx context.Context, msg queue.VideoMessage) error {
	key := "messaage_queue:job_id:" + msg.JobId
	if err := s.cache.Set(ctx, key, "processing", 10*time.Minute); err != nil {
		return err
	}

	gifId, err := s.processor.Process(ctx, msg.JobId, msg.Key, msg.Start, msg.End, msg.Width, msg.FPS, msg.Loop)
	if err != nil {
		return err
	}

	gifKey := "messaage_queue_gif:job_id:" + msg.JobId
	if err := s.cache.Set(ctx, gifKey, gifKey, 5*time.Minute); err != nil {
		return err
	}

	if err := s.cache.Set(ctx, key, "completed", 5*time.Minute); err != nil {
		return err
	}

	gif := media.Gif{
		Key:    gifId,
		UserId: msg.UserID,
		// Url:    w.gifRepo.GetUrl(gifId),
	}

	if err := s.gifRepo.Create(gif); err != nil {
		if err := s.cache.Set(ctx, key, "failed", 5*time.Minute); err != nil {
			return err
		}
		return err
	}
	return nil
}
