package media

import (
	"context"
	"net/url"
	"time"
)

type StorageRepository interface {
	Create(ctx context.Context, key string, expirey time.Duration) (*url.URL, error)
	Download(ctx context.Context, key string, expirey time.Duration) (*url.URL, error)
	IsExists(ctx context.Context, key string) (bool, error)
	Status(ctx context.Context, key string) (Info, error)
	GetObject(ctx context.Context, start, end int64, key string) (Object, error)
	DownloadLocal(ctx context.Context, key, destPath string) error
	Upload(ctx context.Context, key, filePath, contentType string) error
}

type GifRepository interface {
	Create(gif Gif) error
	Get(user_id string, status string) ([]GifResp, error)
	GetByKey(key string) (GifResp, error)
	GetUrl(key string) string
	GetRecents(user_id string) ([]GifResp, error)
	Delete(key string) error
	Update(key string, gif GifResp) error
	SaveRecent(key string) error
}

type LastVideoRepository interface {
	Create(ctx context.Context, upload LastUpload) error
	GetLastVideo(user_id string) (LastUploadResp, error)
}
