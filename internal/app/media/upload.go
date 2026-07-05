package media

import (
	"context"
	"path/filepath"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/random"
)

func (s *service) Upload(rctx context.Context, filename string, contentType string, claims jwt.Payload) (*media.UploadResult, error) {

	userId := claims.Subject
	_ = userId
	_ = contentType

	ext := filepath.Ext(filename)
	key := random.GenerateRandomID().String() + ext
	expirey := 5 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url, err := s.storage.Create(ctx, key, expirey)
	if err != nil {
		return nil, err
	}

	if err := s.UpdateUploadingStatus(rctx, key, "uploading"); err != nil {
		return nil, err
	}

	return &media.UploadResult{
		Url:      url.String(),
		Key:      key,
		ExpireIn: int64(expirey.Seconds()),
	}, nil
}
