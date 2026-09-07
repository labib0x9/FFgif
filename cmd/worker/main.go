package main

import (
	"context"
	"log/slog"
	"os/signal"
	"sync"
	"syscall"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/infra/ffmpeg"
	"github.com/labib0x9/ffgif/internal/infra/mailer"
	"github.com/labib0x9/ffgif/internal/infra/minio"
	"github.com/labib0x9/ffgif/internal/infra/postgres"
	"github.com/labib0x9/ffgif/internal/infra/rabbitmq"
	"github.com/labib0x9/ffgif/internal/infra/redis"
	"github.com/labib0x9/ffgif/internal/infra/redis/cache"
	"github.com/labib0x9/ffgif/internal/worker"

	mediaapp "github.com/labib0x9/ffgif/internal/app/media"
)

func main() {

	cnf := config.GetConfig()

	mailer := mailer.NewSmtpMailer(cnf)

	rabbitMq := rabbitmq.NewRabbitMQ(cnf.RabbitMq)
	defer rabbitMq.Close()

	redisClient := redis.Client(cnf.Redis)
	defer redisClient.Close()

	dbConn := postgres.NewPostgresConn(cnf.PostgreSQL)
	defer dbConn.Close()

	minioClient := minio.NewMinio(cnf.Minio)
	minioPresignedClient := minio.NewPublicMinio(cnf.Minio)

	cacheRepo := cache.NewCache(redisClient)

	storageRepo := minio.NewStorageRepository(minioClient, minioPresignedClient, cnf.Minio)

	authRepo := postgres.NewAuthRepository(dbConn)
	userRepo := postgres.NewUserRepository(dbConn)
	quotaRepo := postgres.NewQuotaRepository(dbConn)
	shareRepo := postgres.NewShareRepository(dbConn)

	lastUploadRepo := postgres.NewLastVideoRepository(dbConn)
	gifRepo := postgres.NewGifRepository(dbConn, cnf.Minio)

	ffmpeg := ffmpeg.NewFmeg(storageRepo)
	tnx := postgres.NewTxManager(dbConn)

	mediaService := mediaapp.NewService(authRepo, userRepo, quotaRepo, gifRepo, shareRepo, lastUploadRepo, storageRepo, tnx, rabbitMq, cacheRepo, ffmpeg, cnf)

	emailWorker := worker.NewEmailWorker(rabbitMq, mailer)
	convertWorker := worker.NewVideoWorker(mediaService, rabbitMq)
	saveMetadataWorker := worker.NewSaveVideoWorker(rabbitMq, mediaService)
	processingWorker := worker.NewProcessingWorker(mediaService, rabbitMq)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(4)

	go func() {
		defer wg.Done()
		emailWorker.Run(ctx, "email-worker", 10)
	}()
	go func() {
		defer wg.Done()
		convertWorker.Run(ctx, "convert-worker", 2)
	}()
	go func() {
		defer wg.Done()
		saveMetadataWorker.Run(ctx, "save-worker", 5)
	}()
	go func() {
		defer wg.Done()
		processingWorker.Run(ctx, "processing-worker", 3)
	}()

	<-ctx.Done()
	slog.Info("shutting down workers...")

	wg.Wait()
	slog.Info("all workers exited cleanly")

}
