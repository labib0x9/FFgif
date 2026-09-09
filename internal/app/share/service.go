package share

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/port/queue"
)

//go:generate mockgen -source=service.go -destination=mocks/mock_share_service.go -package=mocks
type Service interface {
	Create(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error
	CreateByToken(ctx context.Context, sharedBy string, gifKey string, sharedWithEmail string, expiresAt time.Time) (string, error)
	Delete(ctx context.Context, userId, gifKey, shareId string) error
	Get(ctx context.Context, user string) ([]share.GifResponse, error)
	GetByToken(ctx context.Context, token string) (share.GifTokenResponse, error)
}

type service struct {
	authRepo  auth.AuthRepository
	gifRepo   media.GifRepository
	shareRepo share.ShareRepository
	queue     queue.Queue
}

func NewService(
	authRepo auth.AuthRepository,
	gifRepo media.GifRepository,
	shareRepo share.ShareRepository,
	queue queue.Queue,
) Service {
	return &service{
		authRepo:  authRepo,
		gifRepo:   gifRepo,
		shareRepo: shareRepo,
		queue:     queue,
	}
}
