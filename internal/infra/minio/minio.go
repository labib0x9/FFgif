package minio

import (
	"context"
	"log/slog"
	"time"

	"github.com/labib0x9/ffgif/config"
	"github.com/minio/minio-go/v7"
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

func Setup(client *minio.Client, cnf *config.Minio) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := bucketSetup(ctx, client, cnf.TempBucket); err != nil {
		return err
	}
	if err := bucketSetup(ctx, client, cnf.StorageBucket); err != nil {
		return err
	}
	slog.Info("buckets created")

	if err := bindBucket(ctx, client, cnf.TempBucket); err != nil {
		return err
	}
	slog.Info("notification bound")

	if err := setTTLonTempBucket(ctx, client, cnf.TempBucket, cnf.TTL); err != nil {
		return err
	}
	slog.Info("lifecycle set")

	getNotification, err := client.GetBucketNotification(ctx, cnf.TempBucket)
	if err == nil {
		slog.Info("Notification config", "config", getNotification)
	}

	getLifecycle, err := client.GetBucketLifecycle(ctx, cnf.TempBucket)
	if err == nil {
		slog.Info("Lifecycle config", "config", getLifecycle)
	}

	getCors, err := client.GetBucketCors(ctx, cnf.TempBucket)
	if err == nil {
		slog.Info("CORS config", "config", getCors)
	}

	getCors, err = client.GetBucketCors(ctx, cnf.StorageBucket)
	if err == nil {
		slog.Info("CORS config", "config", getCors)
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
		// notification.ObjectCreatedPut,
		// notification.ObjectCreatedCompleteMultipartUpload,
		notification.ObjectCreatedAll,
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
