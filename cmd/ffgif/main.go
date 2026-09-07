package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ffgif/config"
	authapp "github.com/labib0x9/ffgif/internal/app/auth"
	mediaapp "github.com/labib0x9/ffgif/internal/app/media"
	shareapp "github.com/labib0x9/ffgif/internal/app/share"
	userapp "github.com/labib0x9/ffgif/internal/app/user"
	"github.com/labib0x9/ffgif/internal/infra/ffmpeg"
	"github.com/labib0x9/ffgif/internal/infra/minio"
	"github.com/labib0x9/ffgif/internal/infra/postgres"
	"github.com/labib0x9/ffgif/internal/infra/rabbitmq"
	"github.com/labib0x9/ffgif/internal/infra/redis"
	"github.com/labib0x9/ffgif/internal/infra/redis/cache"
	ratelimitter "github.com/labib0x9/ffgif/internal/infra/redis/rate_limiter"
	rest "github.com/labib0x9/ffgif/internal/transport/http"
	authhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/auth"
	mediahandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	sharehandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/static"
	userhandler "github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
)

func main() {
	cnf := config.GetConfig()

	dbConn := postgres.NewPostgresConn(cnf.PostgreSQL)
	defer dbConn.Close()

	redisClient := redis.Client(cnf.Redis)
	defer redisClient.Close()

	minioClient := minio.NewMinio(cnf.Minio)
	minioPresignedClient := minio.NewPublicMinio(cnf.Minio)
	rabbitMq := rabbitmq.NewRabbitMQ(cnf.RabbitMq)
	defer rabbitMq.Close()

	cacheRepo := cache.NewCache(redisClient)
	limiterRepo := ratelimitter.NewRateLimiter(redisClient)

	storageRepo := minio.NewStorageRepository(minioClient, minioPresignedClient, cnf.Minio)

	authRepo := postgres.NewAuthRepository(dbConn)
	// adminRepo := repo.NewAdminRepository(dbConn)
	userRepo := postgres.NewUserRepository(dbConn)
	verifierRepo := postgres.NewVerifierRepo(dbConn)
	reseterRepo := postgres.NewReseterRepo(dbConn)
	quotaRepo := postgres.NewQuotaRepository(dbConn)

	lastUploadRepo := postgres.NewLastVideoRepository(dbConn)
	gifRepo := postgres.NewGifRepository(dbConn, cnf.Minio) // ?? db + bucket
	shareRepo := postgres.NewShareRepository(dbConn)

	jwtProvider := jwt.NewJwt(cnf.JwtSecret)
	hasher := password.NewHasher(cnf.HashPepper, cnf.BcryptCost)
	middlewares := middleware.NewMiddlewares(cnf, cacheRepo, *jwtProvider)
	validate := validator.New()
	ffmpeg := ffmpeg.NewFmeg(storageRepo)

	tnx := postgres.NewTxManager(dbConn)

	authService := authapp.NewService(authRepo, verifierRepo, userRepo, reseterRepo, quotaRepo, cacheRepo, rabbitMq, *jwtProvider, *hasher, tnx)
	mediaService := mediaapp.NewService(authRepo, userRepo, quotaRepo, gifRepo, shareRepo, lastUploadRepo, storageRepo, tnx, rabbitMq, cacheRepo, ffmpeg, cnf)
	shareService := shareapp.NewService(authRepo, gifRepo, shareRepo, rabbitMq)
	userService := userapp.NewService(userRepo, quotaRepo, authRepo, tnx, *jwtProvider, *hasher)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	authHandler := authhandler.NewHandler(authService, middlewares, validate)
	mediaHandler := mediahandler.NewHandler(mediaService, middlewares, validate)
	shareHandler := sharehandler.NewHandler(shareService, middlewares, validate)
	userHandler := userhandler.NewHandler(userService, middlewares, validate)
	staticHandler := static.NewHandler()

	server := rest.NewServer(
		authHandler,
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
