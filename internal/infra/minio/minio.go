package minio

import (
	"context"
	"time"

	"github.com/labib0x9/ffgif/config"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/notification"
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

func NewMinioAdmin(cnf *config.Minio) *madmin.AdminClient {
	admin, err := madmin.NewWithOptions(
		cnf.Endpoint,
		&madmin.Options{
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
	return admin
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

	if err := bindBucket(ctx, client, cnf.TempBucket); err != nil {
		return err
	}

	if err := setTTLonTempBucket(ctx, client, cnf.TempBucket, cnf.TTL); err != nil {
		return err
	}

	if err := setBucketCORS(ctx, client, cnf.TempBucket, cnf.Allowed); err != nil {
		return err
	}

	if err := setBucketCORS(ctx, client, cnf.StorageBucket, cnf.Allowed); err != nil {
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

func bindBucket(ctx context.Context, client *minio.Client, bucket string) error {
	arn := notification.NewArn("minio", "sqs", "", "1", "amqp")

	cfg := notification.NewConfig(arn)
	cfg.AddEvents(
		notification.ObjectCreatedPut,
		notification.ObjectCreatedCompleteMultipartUpload,
	)

	var notificationCfg notification.Configuration
	notificationCfg.AddQueue(cfg)

	return client.SetBucketNotification(ctx, bucket, notificationCfg)
}

func setTTLonTempBucket(ctx context.Context, client *minio.Client, bucket string, TTL int) error {
	days := lifecycle.ExpirationDays(TTL)

	cfg := lifecycle.NewConfiguration()
	cfg.Rules = []lifecycle.Rule{
		{
			ID:     "expire-temp-objects",
			Status: "Enabled",
			Expiration: lifecycle.Expiration{
				Days: days,
			},
		},
	}
	return client.SetBucketLifecycle(ctx, bucket, cfg)
}

func setBucketCORS(ctx context.Context, client *minio.Client, bucket string, origins []string) error {
	CORSRules := []cors.Rule{
		{
			AllowedOrigin: origins,
			AllowedMethod: []string{"GET", "PUT", "HEAD"},
			AllowedHeader: []string{"*"},
			ExposeHeader:  []string{"ETag"},
			MaxAgeSeconds: 3600,
		},
	}
	corsConfig := cors.NewConfig(CORSRules)
	return client.SetBucketCors(ctx, bucket, corsConfig)
}
