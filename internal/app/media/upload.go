package media

import (
	"context"
	"path/filepath"
	"time"

	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/random"
)

type UploadResult struct {
	Url      string `json:"upload_url"`
	Key      string `json:"key"`
	ExpireIn int64  `json:"expires_in"`
}

func (s *service) Upload(filename string, claims jwt.Payload) (*UploadResult, error) {

	userId := claims.Subject
	ext := filepath.Ext(filename)
	key := userId + random.GenerateRandomID().String() + ext
	expirey := 5 * time.Minute

	url, err := s.storage.Create(context.Background(), key, expirey)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		Url:      url.String(),
		Key:      key,
		ExpireIn: int64(expirey.Seconds()),
	}, nil
}
