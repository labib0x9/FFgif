package minio

import (
	"context"
	"net/url"
	"time"

	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/media"
	minio_go "github.com/minio/minio-go/v7"
)

type uploaderRepo struct {
	client *Storage
	cnf    *config.MinioConfig
}

func NewUploaderRepository(client *Storage, cnf *config.MinioConfig) media.UploaderRepository {
	return &uploaderRepo{
		client: client,
		cnf:    cnf,
	}
}

func (u *uploaderRepo) Create(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
	return u.client.PresignedPutObject(ctx, u.cnf.BucketName, key, expirey)
}

func (u *uploaderRepo) Status(ctx context.Context, key string) (bool, error) {
	info, err := u.client.StatObject(ctx, u.cnf.BucketName, key, minio_go.StatObjectOptions{})
	if err != nil {
		return false, err
	}
	if info.Size == 0 {
		return false, err
	}
	return true, nil
}

func (u *uploaderRepo) Delete() error {
	return nil
}

func (u *uploaderRepo) StatObject(ctx context.Context, key string) (media.Info, error) {
	info, err := u.client.StatObject(ctx, u.cnf.BucketName, key, minio_go.StatObjectOptions{})

	return media.Info{
		Size:        info.Size,
		ContentType: info.ContentType,
		UploadedAt:  info.LastModified,
	}, err
}

func (u *uploaderRepo) GetObject(ctx context.Context, start, end int64, key string) (media.Object, error) {
	opts := minio_go.GetObjectOptions{}
	if start != -1 || end != -1 {
		opts.SetRange(start, end)
	}

	obj, err := u.client.GetObject(ctx, u.cnf.BucketName, key, opts)
	return media.Object{obj}, err
}

func (u *uploaderRepo) Download(ctx context.Context, key, destPath string) error {
	return u.client.FGetObject(ctx, u.cnf.BucketName, key, destPath, minio_go.GetObjectOptions{})
}

func (u *uploaderRepo) Upload(ctx context.Context, key, filePath, contentType string) error {
	_, err := u.client.FPutObject(ctx, u.cnf.BucketName, key, filePath,
		minio_go.PutObjectOptions{
			ContentType: contentType,
		},
	)
	return err
}
