package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/labib0x9/ffgif/config"
	mediaapp "github.com/labib0x9/ffgif/internal/app/media"
	"github.com/labib0x9/ffgif/internal/infra/ffmpeg"
	"github.com/labib0x9/ffgif/internal/infra/mailer"
	"github.com/labib0x9/ffgif/internal/infra/minio"
	"github.com/labib0x9/ffgif/internal/infra/postgres"
	"github.com/labib0x9/ffgif/internal/infra/rabbitmq"
	"github.com/labib0x9/ffgif/internal/infra/redis"
	"github.com/labib0x9/ffgif/internal/infra/redis/cache"
	"github.com/labib0x9/ffgif/internal/worker"
	"github.com/labib0x9/ffgif/pkg/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cnf := config.GetConfig()

	// Setup structured telemetry logging
	telemetry.SetupLogger("ffgif-worker", cnf.Telemetry.Environment)

	// Initialize OpenTelemetry tracer provider with OTLP / Jaeger
	shutdownTracer, err := telemetry.InitTracer(
		context.Background(),
		"ffgif-worker",
		cnf.Telemetry.OTLPEndpoint,
		cnf.Telemetry.Environment,
		cnf.Version,
	)
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdownTracer(ctx); err != nil {
				slog.Error("failed to shutdown tracer", "error", err)
			}
		}()
	}

	// Start Worker Prometheus metrics endpoint
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cnf.Telemetry.WorkerMetricsPort),
		Handler: metricsMux,
	}

	go func() {
		slog.Info("Worker metrics server listening", "port", cnf.Telemetry.WorkerMetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Worker metrics server error", "error", err)
		}
	}()

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
	gifRepo := postgres.NewGifRepository(dbConn)

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("error shutting down metrics server", "error", err)
	}

	wg.Wait()
	slog.Info("all workers exited cleanly")
}
