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
	DownloadLocalRawVideo(ctx context.Context, key, destPath string) error
	Upload(ctx context.Context, key, filePath, contentType string) error
	GetStreamURL(ctx context.Context, key string, expiry time.Duration) (*url.URL, error)
}

type GifRepository interface {
	Create(ctx context.Context, gif Gif) error
	Get(ctx context.Context, user_id string, status string) ([]GifResp, error)
	GetByKey(ctx context.Context, key string) (GifResp, error)
	GetRecents(ctx context.Context, user_id string) ([]GifResp, error)
	Delete(ctx context.Context, key string) error
	Update(ctx context.Context, key string, gif GifResp) error
	SaveRecent(ctx context.Context, key string) error
}

type LastVideoRepository interface {
	Create(ctx context.Context, upload LastUpload) error
	GetLastVideo(ctx context.Context, user_id string) (LastUploadResp, error)
}
