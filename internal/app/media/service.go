package media

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/auth"
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
	Status(ctx context.Context, key string) (bool, error)
	Stream(ctx context.Context, key string, Range string) (*StreamResult, error)
	Update(key string) error
	Confirm(key string, filename string, claims jwt.Payload) error
	Upload(filename string, claims jwt.Payload) (*UploadResult, error)
}

type service struct {
	authRepo      auth.AuthRepository
	profileRepo   user.UserRepository
	quotaRepo     user.QuotaRepository
	gifRepo       media.GifRepository
	lastVideoRepo media.LastVideoRepository
	uploaderRepo  media.UploaderRepository
	queue         queue.SaveQueue
}

func NewService(
	authRepo auth.AuthRepository,
	profileRepo user.UserRepository,
	quotaRepo user.QuotaRepository,
	gifRepo media.GifRepository,
	lastVideoRepo media.LastVideoRepository,
	uploaderRepo media.UploaderRepository,
	queue queue.SaveQueue,
) Service {
	return &service{
		authRepo:      authRepo,
		profileRepo:   profileRepo,
		quotaRepo:     quotaRepo,
		gifRepo:       gifRepo,
		lastVideoRepo: lastVideoRepo,
		uploaderRepo:  uploaderRepo,
		queue:         queue,
	}
}

// type Jwt interface {
// 	Create(fullname string, id string, email string, role string) (string, error)
// 	Verify(tokenStr string) (jwt.Payload, error)
// }

// type Hasher interface {
// 	GenerateHash(pass string) (string, error)
// 	CompareHashAndPassword(hashedPass string, pass string) bool
// }
