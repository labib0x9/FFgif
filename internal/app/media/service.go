package media

import (
	"context"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/processor"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/jwt"
)

type Service interface {
	Delete(ctx context.Context, key string) error
	Download(ctx context.Context, key string) (string, error)
	GetByKey(ctx context.Context, key string) (media.GifResp, error)
	GetRecents(ctx context.Context, id string) ([]media.GifResp, error)
	GetGifs(ctx context.Context, id string, filter string) (*GifResult, error)
	LastVideo(ctx context.Context, userId string) (media.LastUploadResp, error)
	Save(ctx context.Context, key string) error
	Stream(ctx context.Context, key string) (*media.StreamResult, error)
	Update(ctx context.Context, key string) error
	Upload(rctx context.Context, filename string, claims jwt.Payload) (*media.UploadResult, error)
	ProcessAndSave(ctx context.Context, key string) error
	UpdateUploadingStatus(ctx context.Context, key string, status string) error
	Status(ctx context.Context, key string) (string, error)
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
