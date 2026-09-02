package media

import (
	"context"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/internal/port/processor"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/jwt"
)

type Service interface {
	Delete(ctx context.Context, userId string, key string) error
	Download(ctx context.Context, key string) (string, error)
	GetByKey(ctx context.Context, key string) (media.GifResponse, error)
	GetRecents(ctx context.Context, id string) ([]media.GifResponse, error)
	GetGifs(ctx context.Context, id string, filter string) (*GifResult, error)
	LastVideo(ctx context.Context, userId string) (media.LastUploadResponse, error)
	Save(ctx context.Context, key string) error
	Stream(ctx context.Context, key string) (*media.StreamResult, error)
	Update(ctx context.Context, key string) error
	Upload(rctx context.Context, filename string, claims jwt.Payload) (*media.UploadResult, error)
	ProcessAndSave(ctx context.Context, key string) error
	UpdateUploadingStatus(ctx context.Context, key string, status string) error
	Status(ctx context.Context, key string) (string, error)

	Process(ctx context.Context, msg queue.VideoMessage) error
	ConversionStatus(ctx context.Context, jobId string) (*StatusResult, error)
	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
	SaveMetadata(ctx context.Context, msg queue.SaveVideoMessage) error
}

type service struct {
	authRepo      auth.AuthRepository
	profileRepo   user.UserRepository
	quotaRepo     user.QuotaRepository
	gifRepo       media.GifRepository
	lastVideoRepo media.LastVideoRepository
	storage       media.StorageRepository
	queue         queue.Queue
	cache         cache.Cache
	processor     processor.VideoProcessor
	cnf           *config.Config
}

func NewService(
	authRepo auth.AuthRepository,
	profileRepo user.UserRepository,
	quotaRepo user.QuotaRepository,
	gifRepo media.GifRepository,
	lastVideoRepo media.LastVideoRepository,
	storage media.StorageRepository,
	queue queue.Queue,
	cache cache.Cache,
	processor processor.VideoProcessor,
	cnf *config.Config,
) Service {
	return &service{
		authRepo:      authRepo,
		profileRepo:   profileRepo,
		quotaRepo:     quotaRepo,
		gifRepo:       gifRepo,
		lastVideoRepo: lastVideoRepo,
		storage:       storage,
		queue:         queue,
		cache:         cache,
		processor:     processor,
		cnf:           cnf,
	}
}

// type Service interface {
// 	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
// 	Status(ctx context.Context, jobId string) (*StatusResult, error)
// 	Process(ctx context.Context, msg queue.VideoMessage) error
// 	SaveMetadata(ctx context.Context, msg queue.SaveVideoMessage) error
// }

// type service struct {
// 	processor     processor.VideoProcessor
// 	gifRepo       media.GifRepository
// 	lastVideoRepo media.LastVideoRepository
// 	minioRepo     media.StorageRepository
// 	cache         cache.Cache
// 	queue         queue.Queue
// }

// func NewService(
// 	processor processor.VideoProcessor,
// 	gifRepo media.GifRepository,
// 	lastVideoRepo media.LastVideoRepository,
// 	cache cache.Cache,
// 	queue queue.Queue,
// ) Service {
// 	return &service{
// 		processor:     processor,
// 		gifRepo:       gifRepo,
// 		lastVideoRepo: lastVideoRepo,
// 		minioRepo:     minioRepo,
// 		cache:         cache,
// 		queue:         queue,
// 	}
// }
