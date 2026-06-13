package auth

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/internal/app/auth"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

type Handler struct {
	srv         auth.Service
	middlewares *middleware.Middlewares
	// authRepo     repo.AuthRepository
	// verifierRepo repo.VerifierRepo
	// cacheRepo    repo.CacheRepo
	// reseterRepo  repo.ReseterRepo
	// userRepo     repo.UserRepository
	// quotaRepo    repo.QuotaRepository
	validate *validator.Validate
	// rabbitMq     *rabbitmq.RabbitMQ
}

func NewHandler(
	srv auth.Service,
	// authRepo repo.AuthRepository,
	// verifierRepo repo.VerifierRepo,
	// cacheRepo repo.CacheRepo,
	// reseterRepo repo.ReseterRepo,
	// userRepo repo.UserRepository,
	// quotaRepo repo.QuotaRepository,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
	// rabbitMq *rabbitmq.RabbitMQ,
) *Handler {
	return &Handler{
		// authRepo:     authRepo,
		// verifierRepo: verifierRepo,
		// cacheRepo:    cacheRepo,
		// reseterRepo:  reseterRepo,
		// userRepo:     userRepo,
		// quotaRepo:    quotaRepo,
		middlewares: middlewares,
		validate:    validate,
		// rabbitMq:     rabbitMq,
		srv: srv,
	}
}
