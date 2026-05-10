package auth

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/repo"
	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

type Handler struct {
	middlewares  *middleware.Middlewares
	authRepo     repo.AuthRepository
	verifierRepo repo.VerifierRepo
	cacheRepo    repo.CacheRepo
	reseterRepo  repo.ReseterRepo
	userRepo     repo.UserRepository
	quotaRepo    repo.QuotaRepository
	validate     *validator.Validate
	rabbitMq     *rabbitmq.RabbitMQ
}

func NewHandler(
	authRepo repo.AuthRepository,
	verifierRepo repo.VerifierRepo,
	cacheRepo repo.CacheRepo,
	reseterRepo repo.ReseterRepo,
	userRepo repo.UserRepository,
	quotaRepo repo.QuotaRepository,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
	rabbitMq *rabbitmq.RabbitMQ,
) *Handler {
	return &Handler{
		authRepo:     authRepo,
		verifierRepo: verifierRepo,
		cacheRepo:    cacheRepo,
		reseterRepo:  reseterRepo,
		userRepo:     userRepo,
		quotaRepo:    quotaRepo,
		middlewares:  middlewares,
		validate:     validate,
		rabbitMq:     rabbitMq,
	}
}
