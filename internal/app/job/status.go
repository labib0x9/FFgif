package job

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/job"
)

type StatusResult struct {
	JobId     string    `json:"job_id"`
	Status    string    `json:"status"`
	GifId     string    `json:"gif_id"`
	Progress  int       `json:"progress"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *service) Status(ctx context.Context, jobId string) (*StatusResult, error) {

	key := "messaage_queue:job_id:" + jobId
	val, err := s.cache.Get(ctx, key)
	if err != nil {
		// key expired..
		// return
		// slog.Warn("Status: Get key failed", "Err", err)
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, job.ErrCacheGetFailed
	}

	gifKey := "messaage_queue_gif:job_id:" + jobId
	gif, _ := s.cache.Get(ctx, gifKey)

	return &StatusResult{
		JobId:  jobId,
		GifId:  gif,
		Status: val,
	}, nil
}
