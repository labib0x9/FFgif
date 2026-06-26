package job

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/domain/queue"
)

type Service interface {
	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
	Status(ctx context.Context, jobId string) (*StatusResult, error)
}

type service struct {
	cache cache.Cache
	queue queue.Queue
}

func NewService(
	cache cache.Cache,
	queue queue.Queue,
) Service {
	return &service{
		cache: cache,
		queue: queue,
	}
}
