package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ffgif/config"
	authapp "github.com/labib0x9/ffgif/internal/app/auth"
	jobapp "github.com/labib0x9/ffgif/internal/app/job"
	mediaapp "github.com/labib0x9/ffgif/internal/app/media"
	shareapp "github.com/labib0x9/ffgif/internal/app/share"
	userapp "github.com/labib0x9/ffgif/internal/app/user"
	"github.com/labib0x9/ffgif/internal/infra/gifprocessor"
	"github.com/labib0x9/ffgif/internal/infra/minio"
	"github.com/labib0x9/ffgif/internal/infra/postgres"
	"github.com/labib0x9/ffgif/internal/infra/rabbitmq"
	"github.com/labib0x9/ffgif/internal/infra/redis"
	"github.com/labib0x9/ffgif/internal/infra/redis/cache"
	ratelimitter "github.com/labib0x9/ffgif/internal/infra/redis/rate_limiter"
	rest "github.com/labib0x9/ffgif/internal/transport/http"
	authhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/auth"
	jobhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/job"
	mediahandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	sharehandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/static"
	userhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/internal/worker"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/mailer"
	"github.com/labib0x9/ffgif/pkg/password"
)

func main() {
	cnf := config.GetConfig()

	dbConn := postgres.NewPostgresConn(cnf.PostgreSQL)
	defer dbConn.Close()

	redisClient := redis.Setup(cnf.RedisConfig)
	defer redisClient.Close()

	minioClient := minio.Setup(cnf.MinioConfig)
	rabbitMq := rabbitmq.NewRabbitMQ(cnf.RabbitMq)
	defer rabbitMq.Close()

	cacheRepo := cache.NewCache(redisClient)
	limiterRepo := ratelimitter.NewRateLimiter(redisClient)

	storageRepo := minio.NewStorageRepository(minioClient, cnf.MinioConfig)

	authRepo := postgres.NewAuthRepository(dbConn)
	// adminRepo := repo.NewAdminRepository(dbConn)
	userRepo := postgres.NewUserRepository(dbConn)
	verifierRepo := postgres.NewVerifierRepo(dbConn)
	reseterRepo := postgres.NewReseterRepo(dbConn)
	quotaRepo := postgres.NewQuotaRepository(dbConn)

	lastUploadRepo := postgres.NewLastVideoRepository(dbConn)
	gifRepo := postgres.NewGifRepository(dbConn, cnf.MinioConfig) // ?? db + bucket
	// shareRepo := postgres.NewShareRepository(dbConn)

	jwtProvider := jwt.NewJwt(cnf.JwtSecret)
	hasher := password.NewHasher(cnf.HashPepper, cnf.BcryptCost)
	middlewares := middleware.NewMiddlewares(cnf, cacheRepo, *jwtProvider)
	validate := validator.New()
	mailer := mailer.NewMailer(cnf)
	ffmpeg := gifprocessor.NewFmeg(storageRepo)

	emailWorker := worker.NewEmailWorker(rabbitMq, mailer)
	convertWorker := worker.NewVideoWorker(rabbitMq, ffmpeg, cacheRepo, gifRepo)
	saveMetadataWorker := worker.NewSaveVideoWorker(rabbitMq, lastUploadRepo, storageRepo)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go emailWorker.Run(ctx, "email-worker", 10)
	go convertWorker.Run(ctx, "convert-worker", 2)
	go saveMetadataWorker.Run(ctx, "save-worker", 5)

	authService := authapp.NewService(authRepo, verifierRepo, userRepo, reseterRepo, quotaRepo, cacheRepo, rabbitMq, *jwtProvider, *hasher)
	jobService := jobapp.NewService(cacheRepo, rabbitMq)
	mediaService := mediaapp.NewService(authRepo, userRepo, quotaRepo, gifRepo, lastUploadRepo, storageRepo, rabbitMq)
	shareService := shareapp.NewService()
	userService := userapp.NewService(userRepo, quotaRepo, authRepo, *jwtProvider, *hasher)

	authHandler := authhandler.NewHandler(authService, middlewares, validate)
	jobHandler := jobhandler.NewHandler(jobService, middlewares, validate)
	mediaHandler := mediahandler.NewHandler(mediaService, middlewares, validate)
	shareHandler := sharehandler.NewHandler(shareService, middlewares, validate)
	userHandler := userhandler.NewHandler(userService, middlewares, validate)
	staticHandler := static.NewHandler()

	server := rest.NewServer(
		authHandler,
		jobHandler,
		mediaHandler,
		shareHandler,
		userHandler,
		staticHandler,
	)

	go func() {
		server.Start(limiterRepo, cnf)
	}()

	<-ctx.Done()

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server.Shutdown(shutdown)
}
