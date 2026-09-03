package share

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/share"
)

type Service interface {
	Create(ctx context.Context, sharedBy string, gifKey string, sharedWith string, expiresAt time.Time) error
	Delete(ctx context.Context, userId, gifKey, shareId string) error
	Get(ctx context.Context, user string) ([]share.GifResponse, error)
}

type service struct {
	authRepo  auth.AuthRepository
	gifRepo   media.GifRepository
	shareRepo share.ShareRepository
}

func NewService(
	authRepo auth.AuthRepository,
	gifRepo media.GifRepository,
	shareRepo share.ShareRepository,
) Service {
	return &service{
		authRepo:  authRepo,
		gifRepo:   gifRepo,
		shareRepo: shareRepo,
	}
}
