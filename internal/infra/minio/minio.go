package minio

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinio(cnf *config.MinioConfig) *minio.Client {
	client, err := minio.New(
		cnf.Endpoint,
		&minio.Options{
			Creds: credentials.NewStaticV4(
				cnf.AccessKeyID,
				cnf.SecretAccessKey,
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

func Setup(cnf *config.MinioConfig) *minio.Client {
	client := NewMinio(cnf)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	exist, err := client.BucketExists(ctx, cnf.BucketName)
	if err != nil {
		panic(err)
	}

	if !exist {
		if err := client.MakeBucket(ctx, cnf.BucketName, minio.MakeBucketOptions{}); err != nil {
			panic(err)
		}
	}

	return client
}
