package minio

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinio(cnf *config.Minio) *minio.Client {
	client, err := minio.New(
		cnf.Endpoint,
		&minio.Options{
			Creds: credentials.NewStaticV4(
				cnf.RootUser,
				cnf.RootPass,
				"",
			),
			Secure: false,
		},
	)
	if err != nil {
		panic(err)
	}
	return client
}

func Setup(client *minio.Client, cnf *config.Minio) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := bucketSetup(ctx, client, cnf.TempBucket); err != nil {
		return err
	}

	if err := bucketSetup(ctx, client, cnf.StorageBucket); err != nil {
		return err
	}

	return nil
}

func bucketSetup(ctx context.Context, client *minio.Client, name string) error {
	exist, err := client.BucketExists(ctx, name)
	if err != nil {
		return err
	}

	if !exist {
		if err := client.MakeBucket(ctx, name, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	return nil
}
