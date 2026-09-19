package media

import (
	"context"
	"sync"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/internal/port/db"
	"github.com/labib0x9/ffgif/internal/port/processor"
	"github.com/labib0x9/ffgif/internal/port/queue"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_media_service.go -package=mocks
type Service interface {
	Delete(ctx context.Context, userId string, key string) error
	Download(ctx context.Context, userId, key string) (string, error)
	GetByKey(ctx context.Context, userId string, key string) (*media.GifResponse, error)
	GetRecents(ctx context.Context, userId string) ([]media.GifResponse, error)
	GetGifs(ctx context.Context, userId string, filter string, page int, limit int) (*GifResult, error)
	LastVideo(ctx context.Context, userId string) (media.LastUploadResponse, error)
	Save(ctx context.Context, userId, key string) error
	Stream(ctx context.Context, userId string, key string) (*media.StreamResult, error)
	Update(ctx context.Context, userId string, key string, _gif media.GifUpdateRequest, lastUpdatedAt string) (*media.GifResponse, error)
	Upload(rctx context.Context, filename string, userId string) (*media.UploadResult, error)
	ProcessAndSave(ctx context.Context, key string) error
	UpdateUploadingStatus(ctx context.Context, key string, status string) error
	Status(ctx context.Context, userId, key string) (string, string, error)

	Process(ctx context.Context, msg queue.VideoMessage) error
	ConversionStatus(ctx context.Context, userId string, jobId string) (*StatusResult, error)
	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
	SaveMetadata(ctx context.Context, msg queue.SaveVideoMessage) error

	GetGifThumbnail(ctx context.Context, userId string, key string) (string, error)
}

type service struct {
	quotaRepo     user.QuotaRepository
	gifRepo       media.GifRepository
	shareRepo     share.ShareRepository
	lastVideoRepo media.LastVideoRepository
	jobRepo       media.JobRepository
	storage       media.StorageRepository
	tnx           db.TxManager
	queue         queue.Queue
	cache         cache.Cache
	processor     processor.VideoProcessor
	cnf           *config.Config
	userConvertMu sync.Map
}

func NewService(
	quotaRepo user.QuotaRepository,
	gifRepo media.GifRepository,
	shareRepo share.ShareRepository,
	lastVideoRepo media.LastVideoRepository,
	jobRepo media.JobRepository,
	storage media.StorageRepository,
	tnx db.TxManager,
	queue queue.Queue,
	cache cache.Cache,
	processor processor.VideoProcessor,
	cnf *config.Config,
) Service {
	return &service{
		quotaRepo:     quotaRepo,
		gifRepo:       gifRepo,
		shareRepo:     shareRepo,
		lastVideoRepo: lastVideoRepo,
		jobRepo:       jobRepo,
		storage:       storage,
		tnx:           tnx,
		queue:         queue,
		cache:         cache,
		processor:     processor,
		cnf:           cnf,
	}
}
