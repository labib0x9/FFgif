package media

import (
	"context"
	"errors"
	"net/url"
	"time"

	minio_go "github.com/minio/minio-go/v7"
)

var (
	ErrDeleteByKeyFailed = errors.New("")

	ErrStatFetchFailed         = errors.New("")
	ErrRangeParserFailed       = errors.New("")
	ErrObjectFetchFailed       = errors.New("")
	ErrContentLengthMismatched = errors.New("Content length must be 512byte")
	ErrInvalidExt              = errors.New("invalid file type")
	ErrEmptyKey                = errors.New("key is empty")
	ErrInvalidFiletype         = errors.New("Invalid file")
)

var (
	ContentLength = 512
)

type Info struct {
	Size        int64
	ContentType string
	UploadedAt  time.Time
}

type Object struct {
	*minio_go.Object
}

type UploadResult struct {
	Url      string `json:"upload_url"`
	Key      string `json:"key"`
	ExpireIn int64  `json:"expires_in"`
}

type StreamResult struct {
	PresignedUrl string `json:"url"`
	ExpireIn     int    `json:"expires_in"`
}

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
