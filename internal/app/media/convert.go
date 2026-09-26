package media

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/apperr"
	"github.com/labib0x9/ffgif/pkg/random"
)

type ConvertResult struct {
	Id     string
	Status string
}

func (s *service) Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error) {
	if strings.Contains(key, ":") && !strings.HasPrefix(key, userId+":") {
		return nil, media.ErrGifOwnerMismatch
	}

	val, _ := s.userConvertMu.LoadOrStore(userId, &sync.Mutex{})
	userMu := val.(*sync.Mutex)
	userMu.Lock()
	defer userMu.Unlock()

	quota, err := s.quotaRepo.GetById(ctx, userId)
	if err == nil && quota != nil {
		if quota.GifCount > 0 && quota.GifCount >= quota.GitLimit {
			return nil, user.ErrQuotaExceeded
		}
		quota.GifCount++
	}

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

	ownerKey := "messaage_queue_owner:job_id:" + Id
	if err := s.cache.Set(ctx, ownerKey, userId, 5*time.Minute); err != nil {
		return nil, fmt.Errorf("cache.Set(owner): %w: %w", apperr.ErrCacheSetFailed, err)
	}

	mqkey := "messaage_queue:job_id:" + Id
	if err := s.cache.Set(ctx, mqkey, status, 5*time.Minute); err != nil {
		return nil, fmt.Errorf("cache.Set(job): %w: %w", apperr.ErrCacheSetFailed, err)
	}

	gifKey := "messaage_queue_gif:job_id:" + Id
	if err := s.cache.Set(ctx, gifKey, status, 5*time.Minute); err != nil {
		return nil, fmt.Errorf("cache.Set(gif): %w: %w", apperr.ErrCacheSetFailed, err)
	}

	if s.jobRepo != nil {
		job := media.Job{
			ID:        Id,
			UserID:    userId,
			Type:      "video_to_gif",
			Status:    media.StatusQueued,
			Progress:  0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = s.jobRepo.Create(ctx, job)
	}

	if err := s.queue.PublishVideo(ctx, msg); err != nil {
		_ = s.cache.Set(ctx, mqkey, "failed", 5*time.Minute)
		_ = s.cache.Set(ctx, gifKey, "failed", 5*time.Minute)
		if s.jobRepo != nil {
			_ = s.jobRepo.UpdateStatus(ctx, Id, media.StatusFailed, 0, "", "queue publish failed")
		}
		return nil, fmt.Errorf("queue.PublishVideo: %w: %w", apperr.ErrMessageQueueFailed, err)
	}

	return &ConvertResult{
		Id:     Id,
		Status: status,
	}, nil
}
