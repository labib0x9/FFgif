package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/config"
	authapp "github.com/labib0x9/ProjectUnsafe/internal/app/auth"
	jobapp "github.com/labib0x9/ProjectUnsafe/internal/app/job"
	mediaapp "github.com/labib0x9/ProjectUnsafe/internal/app/media"
	shareapp "github.com/labib0x9/ProjectUnsafe/internal/app/share"
	userapp "github.com/labib0x9/ProjectUnsafe/internal/app/user"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/gifprocessor"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/minio"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/postgres"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/redis"
	rest "github.com/labib0x9/ProjectUnsafe/internal/transport/http"
	authhandler "github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/auth"
	jobhandler "github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/job"
	mediahandler "github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/media"
	sharehandler "github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/share"
	userhandler "github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/user"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
	"github.com/labib0x9/ProjectUnsafe/internal/worker"
	"github.com/labib0x9/ProjectUnsafe/pkg/jwt"
	"github.com/labib0x9/ProjectUnsafe/pkg/mailer"
	"github.com/labib0x9/ProjectUnsafe/pkg/password"
)

func main() {
	cnf := config.GetConfig()

	postgresConn := postgres.New()
	dbConn := postgresConn.SetupAndConnection(cnf.DBConfig)
	defer dbConn.Close()

	redisClient := redis.Setup(cnf.RedisConfig)
	defer redisClient.Close()

	minioClient := minio.Setup(cnf.MinioConfig)
	rabbitMq := rabbitmq.NewRabbitMQ()
	defer rabbitMq.Close()

	authRepo := postgres.NewAuthRepository(dbConn)
	cacheRepo := redis.NewCacheRepo(redisClient)
	uploaderRepo := minio.NewUploaderRepository(&minioClient, cnf.MinioConfig)
	_ = uploaderRepo

	// adminRepo := repo.NewAdminRepository(dbConn)
	userRepo := postgres.NewUserRepository(dbConn)
	verifierRepo := postgres.NewVerifierRepo(dbConn)
	reseterRepo := postgres.NewReseterRepo(dbConn)
	quotaRepo := postgres.NewQuotaRepository(dbConn)

	lastUploadRepo := postgres.NewLastVideoRepository(dbConn)
	if lastUploadRepo == nil {
		slog.Error("LAST UPLOAD NIL")
	}
	gifRepo := postgres.NewGifRepository(dbConn, cnf.MinioConfig) // ?? db + bucket
	// shareRepo := postgres.NewShareRepository(dbConn)

	jwtProvider := jwt.NewJwt(cnf.JwtSecret)
	hasher := password.NewHasher(cnf.HashPepper, cnf.BcryptCost)
	middlewares := middleware.NewMiddlewares(cnf, cacheRepo, *jwtProvider)
	validate := validator.New()
	mailer := mailer.NewMailer(cnf)
	ffmpeg := gifprocessor.NewFmeg(uploaderRepo)

	emailWorker := worker.NewEmailWorker(rabbitMq, mailer)
	convertWorker := worker.NewVideoWorker(rabbitMq, ffmpeg, cacheRepo, gifRepo)
	saveMetadataWorker := worker.NewSaveVideoWorker(rabbitMq, lastUploadRepo, uploaderRepo)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go emailWorker.Run(ctx, 10)
	go convertWorker.Run(ctx, 2)
	go saveMetadataWorker.Run(ctx, 5)

	authService := authapp.NewService(authRepo, verifierRepo, userRepo, reseterRepo, quotaRepo, cacheRepo, rabbitMq, *jwtProvider, *hasher)
	jobService := jobapp.NewService(cacheRepo, rabbitMq)
	mediaService := mediaapp.NewService(authRepo, userRepo, quotaRepo, gifRepo, lastUploadRepo, uploaderRepo, rabbitMq)
	shareService := shareapp.NewService()
	userService := userapp.NewService(userRepo, quotaRepo, authRepo, *jwtProvider, *hasher)

	authHandler := authhandler.NewHandler(authService, middlewares, validate)
	jobHandler := jobhandler.NewHandler(jobService, middlewares, validate)
	mediaHandler := mediahandler.NewHandler(mediaService, middlewares, validate)
	shareHandler := sharehandler.NewHandler(shareService, middlewares, validate)
	userHandler := userhandler.NewHandler(userService, middlewares, validate)

	server := rest.NewServer(
		authHandler,
		jobHandler,
		mediaHandler,
		shareHandler,
		userHandler,
	)

	go func() {
		server.Start(redisClient, cnf)
	}()

	<-ctx.Done()

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server.Shutdown(shutdown)
}
