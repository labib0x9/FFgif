package repo

import (
	"context"
	"net/url"
	"time"

	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/infra/minio"
	minio_go "github.com/minio/minio-go/v7"
)

type Info struct {
	Size        int64
	ContentType string
}

type Object struct {
	*minio_go.Object
}

// func (o *Object) Close() {
// 	o.Close()
// }

type UploaderRepository interface {
	Create(ctx context.Context, key string, expirey time.Duration) (*url.URL, error)
	Status(ctx context.Context, key string) (bool, error)
	Delete() error
	StatObject(ctx context.Context, key string) (Info, error)
	GetObject(ctx context.Context, start, end int64, key string) (Object, error)
}

type uploaderRepo struct {
	client *minio.Storage
	cnf    *config.MinioConfig
}

func NewUploaderRepository(client *minio.Storage, cnf *config.MinioConfig) UploaderRepository {
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

func (u *uploaderRepo) StatObject(ctx context.Context, key string) (Info, error) {
	info, err := u.client.StatObject(ctx, u.cnf.BucketName, key, minio_go.StatObjectOptions{})

	return Info{
		Size:        info.Size,
		ContentType: info.ContentType,
	}, err
}

func (u *uploaderRepo) GetObject(ctx context.Context, start, end int64, key string) (Object, error) {
	opts := minio_go.GetObjectOptions{}
	opts.SetRange(start, end)

	obj, err := u.client.GetObject(ctx, u.cnf.BucketName, key, opts)
	return Object{obj}, err
}
