package media

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/pkg/apperr"
)

type StatusResult struct {
	JobId     string    `json:"job_id"`
	Status    string    `json:"status"`
	GifId     string    `json:"gif_id"`
	Progress  int       `json:"progress"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *service) ConversionStatus(ctx context.Context, userId string, jobId string) (*StatusResult, error) {
	ownerKey := "messaage_queue_owner:job_id:" + jobId
	owner, err := s.cache.Get(ctx, ownerKey)
	if err == nil {
		if owner != userId {
			return nil, media.ErrGifOwnerMismatch
		}

		key := "messaage_queue:job_id:" + jobId
		val, err := s.cache.Get(ctx, key)
		if err == nil {
			gifKey := "messaage_queue_gif:job_id:" + jobId
			gif, _ := s.cache.Get(ctx, gifKey)
			return &StatusResult{
				JobId:  jobId,
				GifId:  gif,
				Status: val,
			}, nil
		}
	}

	// Fallback to PostgreSQL jobRepo if cache missed
	if s.jobRepo != nil {
		job, err := s.jobRepo.GetByID(ctx, jobId)
		if err == nil && job != nil {
			if job.UserID != userId {
				return nil, media.ErrGifOwnerMismatch
			}
			return &StatusResult{
				JobId:     job.ID,
				Status:    job.Status,
				GifId:     job.ResultKey,
				Progress:  job.Progress,
				UpdatedAt: job.UpdatedAt,
			}, nil
		}
	}

	if err != nil {
		if errors.Is(err, cache.ErrCacheMiss) {
			return nil, fmt.Errorf("cache.Get(owner): %w", apperr.ErrCacheGetFailed)
		}
		return nil, fmt.Errorf("cache.Get(owner): %w", err)
	}

	return nil, fmt.Errorf("cache.Get: %w", apperr.ErrCacheGetFailed)
}
