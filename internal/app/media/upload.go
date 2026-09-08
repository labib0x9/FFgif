package media

import (
	"context"
	"path/filepath"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/pkg/random"
)

// key = <userId>:<uuid>.<ext>
func (s *service) Upload(rctx context.Context, filename string, userId string) (*media.UploadResult, error) {
	ext := filepath.Ext(filename)
	key := userId + ":" + random.GenerateRandomID().String() + ext
	expirey := 5 * time.Minute

	ctx, cancel := context.WithTimeout(rctx, 30*time.Second)
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
