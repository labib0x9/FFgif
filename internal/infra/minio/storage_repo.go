package minio

import (
	"context"
	"net/url"
	"time"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/minio/minio-go/v7"
	minio_go "github.com/minio/minio-go/v7"
)

type storageRepo struct {
	client *minio.Client
	cnf    *config.MinioConfig
}

func NewStorageRepository(client *minio.Client, cnf *config.MinioConfig) media.StorageRepository {
	return &storageRepo{
		client: client,
		cnf:    cnf,
	}
}

// create url, directly upload
func (u *storageRepo) Create(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
	return u.client.PresignedPutObject(ctx, u.cnf.BucketName, key, expirey)
}

func (u *storageRepo) Download(ctx context.Context, key string, expirey time.Duration) (*url.URL, error) {
	values := url.Values{}
	values.Add("response-content-disposition", "attachment; filename="+key)
	return u.client.PresignedGetObject(ctx, u.cnf.BucketName, key, expirey, values)
}

func (u *storageRepo) IsExists(ctx context.Context, key string) (bool, error) {
	info, err := u.client.StatObject(ctx, u.cnf.BucketName, key, minio_go.StatObjectOptions{})
	if err != nil {
		return false, err
	}
	if info.Size == 0 {
		return false, err
	}
	return true, nil
}

func (u *storageRepo) Delete() error {
	return nil
}

func (u *storageRepo) Status(ctx context.Context, key string) (media.Info, error) {
	info, err := u.client.StatObject(ctx, u.cnf.BucketName, key, minio_go.StatObjectOptions{})

	return media.Info{
		Size:        info.Size,
		ContentType: info.ContentType,
		UploadedAt:  info.LastModified,
	}, err
}

func (u *storageRepo) GetObject(ctx context.Context, start, end int64, key string) (media.Object, error) {
	opts := minio_go.GetObjectOptions{}
	if start != -1 || end != -1 {
		opts.SetRange(start, end)
	}

	obj, err := u.client.GetObject(ctx, u.cnf.BucketName, key, opts)
	return media.Object{obj}, err
}

func (u *storageRepo) DownloadLocal(ctx context.Context, key, destPath string) error {
	return u.client.FGetObject(ctx, u.cnf.BucketName, key, destPath, minio_go.GetObjectOptions{})
}

func (u *storageRepo) Upload(ctx context.Context, key, filePath, contentType string) error {
	_, err := u.client.FPutObject(ctx, u.cnf.BucketName, key, filePath,
		minio_go.PutObjectOptions{
			ContentType: contentType,
		},
	)
	return err
}
