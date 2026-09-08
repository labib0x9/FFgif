//go:generate mockgen -package=mocks -destination=mocks/mock_media_service.go github.com/labib0x9/ffgif/internal/app/media Service

package media

import (
	"context"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/internal/port/db"
	"github.com/labib0x9/ffgif/internal/port/processor"
	"github.com/labib0x9/ffgif/internal/port/queue"
)

type Service interface {
	Delete(ctx context.Context, userId string, key string) error
	Download(ctx context.Context, userId, key string) (string, error)
	GetByKey(ctx context.Context, userId string, key string) (*media.GifResponse, error)
	GetRecents(ctx context.Context, userId string) ([]media.GifResponse, error)
	GetGifs(ctx context.Context, userId string, filter string) (*GifResult, error)
	LastVideo(ctx context.Context, userId string) (media.LastUploadResponse, error)
	Save(ctx context.Context, userId, key string) error
	Stream(ctx context.Context, userId string, key string) (*media.StreamResult, error)
	Update(ctx context.Context, userId string, key string, _gif media.GifUpdateRequest, lastUpdatedAt string) (*media.GifResponse, error)
	Upload(rctx context.Context, filename string, userId string) (*media.UploadResult, error)
	ProcessAndSave(ctx context.Context, key string) error
	UpdateUploadingStatus(ctx context.Context, key string, status string) error
	Status(ctx context.Context, userId, key string) (string, string, error)

	Process(ctx context.Context, msg queue.VideoMessage) error
	ConversionStatus(ctx context.Context, jobId string) (*StatusResult, error)
	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
	SaveMetadata(ctx context.Context, msg queue.SaveVideoMessage) error

	GetGifThumbnail(ctx context.Context, userId string, key string) (string, error)
}

type service struct {
	authRepo      auth.AuthRepository
	profileRepo   user.UserRepository
	quotaRepo     user.QuotaRepository
	gifRepo       media.GifRepository
	shareRepo     share.ShareRepository
	lastVideoRepo media.LastVideoRepository
	storage       media.StorageRepository
	tnx           db.TxManager
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
	shareRepo share.ShareRepository,
	lastVideoRepo media.LastVideoRepository,
	storage media.StorageRepository,
	tnx db.TxManager,
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
		shareRepo:     shareRepo,
		lastVideoRepo: lastVideoRepo,
		storage:       storage,
		tnx:           tnx,
		queue:         queue,
		cache:         cache,
		processor:     processor,
		cnf:           cnf,
	}
}
