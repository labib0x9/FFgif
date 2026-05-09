package cmd

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/infra/cache/redis"
	"github.com/labib0x9/ProjectUnsafe/infra/db/postgres"
	"github.com/labib0x9/ProjectUnsafe/infra/minio"
	"github.com/labib0x9/ProjectUnsafe/repo"
	"github.com/labib0x9/ProjectUnsafe/rest"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/admin"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/auth"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/converter"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/uploader"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/user"
	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
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

	authHandler := auth.NewHandler(authRepo, verifierRepo, cacheRepo, reseterRepo, userRepo, quotaRepo, middlewares, validate, mailer)
	adminHandler := admin.NewHandler(adminRepo, middlewares)
	userHandler := user.NewHandler(userRepo, quotaRepo, middlewares, validate)
	uploaderHandler := uploader.NewHandler(uploaderRepo, validate, middlewares)
	converterHandler := converter.NewHandler(cacheRepo, validate, middlewares)

	server := rest.NewServer(
		authHandler,
		adminHandler,
		userHandler,
		uploaderHandler,
		converterHandler,
	)

	server.Start(redisClient, cnf)
}
