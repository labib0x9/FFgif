package media

import (
	"context"
	"path/filepath"
	"time"

	"github.com/labib0x9/ProjectUnsafe/pkg/jwt"
	"github.com/labib0x9/ProjectUnsafe/pkg/random"
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

	url, err := s.uploaderRepo.Create(context.Background(), key, expirey)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Warn("Upload: presigned url create error", "error", err)
		return nil, err
	}

	return &UploadResult{
		Url:      url.String(),
		Key:      key,
		ExpireIn: int64(expirey.Seconds()),
	}, nil
}
