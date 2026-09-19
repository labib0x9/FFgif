package media

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
)

func (s *service) Process(ctx context.Context, msg queue.VideoMessage) error {
	key := "messaage_queue:job_id:" + msg.JobId
	if err := s.cache.Set(ctx, key, "processing", 10*time.Minute); err != nil {
		return err
	}
	if s.jobRepo != nil {
		_ = s.jobRepo.UpdateStatus(ctx, msg.JobId, media.StatusProcessing, 20, "", "")
	}

	result, err := s.processor.Process(ctx, msg.JobId, msg.Key, msg.Start, msg.End, msg.Width, msg.FPS, msg.Loop)
	if err != nil {
		if s.jobRepo != nil {
			_ = s.jobRepo.UpdateStatus(ctx, msg.JobId, media.StatusFailed, 0, "", err.Error())
		}
		return errors.Join(err, s.cache.Set(ctx, key, "failed", 5*time.Minute))
	}

	gifKey := "messaage_queue_gif:job_id:" + msg.JobId
	if err := s.cache.Set(ctx, gifKey, result.GifKey, 5*time.Minute); err != nil {
		return err
	}

	if err := s.cache.Set(ctx, key, "completed", 5*time.Minute); err != nil {
		return err
	}
	if s.jobRepo != nil {
		_ = s.jobRepo.UpdateStatus(ctx, msg.JobId, media.StatusDone, 100, result.GifKey, "")
	}

	gif := media.Gif{
		Key:          result.GifKey,
		UserId:       msg.UserID,
		Url:          result.GifKey,
		ThumbnailUrl: result.ThumbKey,
	}

	if err := s.gifRepo.Create(ctx, gif); err != nil {
		return errors.Join(err, s.cache.Set(ctx, key, "failed", 5*time.Minute))
	}

	// TODO: wire actual gif file size once ffprobe metadata enrichment lands
	ok, err := s.quotaRepo.IncrementUsage(ctx, msg.UserID, 0, 1)
	if err != nil {
		return errors.Join(err, s.cache.Set(ctx, key, "failed", 5*time.Minute))
	}
	if !ok {
		slog.Warn("quota exceeded, gif created but usage not recorded", "job_id", msg.JobId, "user_id", msg.UserID)
	}

	return nil
}
