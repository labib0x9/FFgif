package cmd

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/infra/cache/redis"
	"github.com/labib0x9/ProjectUnsafe/infra/db/postgres"
	"github.com/labib0x9/ProjectUnsafe/infra/minio"
	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/infra/worker"
	"github.com/labib0x9/ProjectUnsafe/repo"
	"github.com/labib0x9/ProjectUnsafe/rest"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/admin"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/auth"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/converter"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/uploader"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/user"
	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils/ffmpeg"
	"github.com/labib0x9/ProjectUnsafe/utils/mailer"
)

func Serve() {
	cnf := config.GetConfig()

	postgresConn := postgres.New()
	dbConn := postgresConn.SetupAndConnection(cnf.DBConfig)
	defer dbConn.Close()

	redisClient := redis.Setup(cnf.RedisConfig)
	defer redisClient.Close()

	minioClient := minio.Setup(cnf.MinioConfig)
	rabbitMq := rabbitmq.NewRabbitMQ()
	defer rabbitMq.Close()

	authRepo := repo.NewAuthRepository(dbConn)
	adminRepo := repo.NewAdminRepository(dbConn)
	userRepo := repo.NewUserRepository(dbConn)
	verifierRepo := repo.NewVerifierRepo(dbConn)
	cacheRepo := repo.NewCacheRepo(redisClient)
	reseterRepo := repo.NewReseterRepo(dbConn)
	uploaderRepo := repo.NewUploaderRepository(&minioClient, cnf.MinioConfig)
	quotaRepo := repo.NewQuotaRepository(dbConn)

	middlewares := middleware.NewMiddlewares(cnf, cacheRepo)
	validate := validator.New()
	mailer := mailer.NewMailer(cnf)
	fmeg := ffmpeg.NewFmeg()

	emailWorker := worker.NewEmailWorker(rabbitMq, mailer)
	convertWorker := worker.NewVideoWorker(rabbitMq, fmeg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go emailWorker.Run(ctx, 10)
	go convertWorker.Run(ctx, 2)

	authHandler := auth.NewHandler(authRepo, verifierRepo, cacheRepo, reseterRepo, userRepo, quotaRepo, middlewares, validate, rabbitMq)
	adminHandler := admin.NewHandler(adminRepo, middlewares)
	userHandler := user.NewHandler(userRepo, quotaRepo, authRepo, middlewares, validate)
	uploaderHandler := uploader.NewHandler(uploaderRepo, validate, middlewares)
	converterHandler := converter.NewHandler(cacheRepo, validate, middlewares, rabbitMq)

	server := rest.NewServer(
		authHandler,
		adminHandler,
		userHandler,
		uploaderHandler,
		converterHandler,
	)

	server.Start(redisClient, cnf)
}
