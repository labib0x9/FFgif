package media

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/apperr"
	"github.com/labib0x9/ffgif/pkg/random"
)

type ConvertResult struct {
	Id     string
	Status string
}

func (s *service) Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error) {
	Id := random.GenerateRandomID().String()
	status := "queued"

	msg := queue.VideoMessage{
		UserID:  userId,
		JobId:   Id,
		Key:     key,
		Start:   start,
		End:     end,
		Width:   width,
		FPS:     fps,
		Loop:    loop,
		Retries: 0,
	}

	mqkey := "messaage_queue:job_id:" + Id
	if err := s.cache.Set(ctx, mqkey, status, 5*time.Minute); err != nil {
		return nil, apperr.ErrCacheSetFailed
	}

	gifKey := "messaage_queue_gif:job_id:" + Id
	if err := s.cache.Set(ctx, gifKey, status, 5*time.Minute); err != nil {
		return nil, apperr.ErrCacheSetFailed
	}

	if err := s.queue.PublishVideo(ctx, msg); err != nil {
		return nil, apperr.ErrMessageQueueFailed
	}

	return &ConvertResult{
		Id:     Id,
		Status: status,
	}, nil
}
