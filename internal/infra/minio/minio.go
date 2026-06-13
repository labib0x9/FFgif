package minio

import (
	"context"

	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	*minio.Client
}

func NewStorage(cnf *config.MinioConfig) Storage {
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
	return Storage{
		client,
	}
}

func Setup(cnf *config.MinioConfig) Storage {
	client := NewStorage(cnf)

	exist, err := client.BucketExists(context.Background(), cnf.BucketName)
	if err != nil {
		panic(err)
	}

	if !exist {
		if err := client.MakeBucket(context.Background(), cnf.BucketName, minio.MakeBucketOptions{}); err != nil {
			panic(err)
		}
	}

	return client
}
