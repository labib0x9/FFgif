package media

import (
	"context"
	"fmt"
	"time"

	"github.com/labib0x9/ffgif/pkg/apperr"
)

type StatusResult struct {
	JobId     string    `json:"job_id"`
	Status    string    `json:"status"`
	GifId     string    `json:"gif_id"`
	Progress  int       `json:"progress"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *service) ConversionStatus(ctx context.Context, jobId string) (*StatusResult, error) {

	key := "messaage_queue:job_id:" + jobId
	val, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("cache.Get: %w: %w", apperr.ErrCacheGetFailed, err)
	}

	gifKey := "messaage_queue_gif:job_id:" + jobId
	gif, err := s.cache.Get(ctx, gifKey)
	if err != nil {
		return nil, err
	}

	return &StatusResult{
		JobId:  jobId,
		GifId:  gif,
		Status: val,
	}, nil
}
