package job

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/processor"
	"github.com/labib0x9/ffgif/internal/domain/queue"
)

type Service interface {
	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
	Status(ctx context.Context, jobId string) (*StatusResult, error)
	Process(ctx context.Context, msg queue.VideoMessage) error
	SaveMetadata(ctx context.Context, msg queue.SaveVideoMessage) error
}

type service struct {
	processor     processor.VideoProcessor
	gifRepo       media.GifRepository
	lastVideoRepo media.LastVideoRepository
	minioRepo     media.StorageRepository
	cache         cache.Cache
	queue         queue.Queue
}

func NewService(
	processor processor.VideoProcessor,
	gifRepo media.GifRepository,
	lastVideoRepo media.LastVideoRepository,
	minioRepo media.StorageRepository,
	cache cache.Cache,
	queue queue.Queue,
) Service {
	return &service{
		processor:     processor,
		gifRepo:       gifRepo,
		lastVideoRepo: lastVideoRepo,
		minioRepo:     minioRepo,
		cache:         cache,
		queue:         queue,
	}
}
