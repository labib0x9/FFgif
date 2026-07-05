package media

import (
	"context"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/jwt"
)

type Service interface {
	Delete(key string) error
	Download(key string) (string, error)
	GetByKey(key string) (media.GifResp, error)
	GetRecents(id string) ([]media.GifResp, error)
	GetGifs(id string, filter string) (*GifResult, error)
	LastVideo(userId string) (media.LastUploadResp, error)
	Save(key string) error
	Stream(ctx context.Context, key string, Range string) (*StreamResult, error)
	Update(key string) error
	Upload(rctx context.Context, filename string, contentType string, claims jwt.Payload) (*media.UploadResult, error)
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
		cnf:           cnf,
	}
}
