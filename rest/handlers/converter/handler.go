package converter

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/repo"
	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

type Handler struct {
	middlewares *middleware.Middlewares
	cacheRepo   repo.CacheRepo
	validate    *validator.Validate
	rabbitMq    *rabbitmq.RabbitMQ
}

func NewHandler(
	cacheRepo repo.CacheRepo,
	validate *validator.Validate,
	middlewares *middleware.Middlewares,
	rabbitMq *rabbitmq.RabbitMQ,
) *Handler {
	return &Handler{
		cacheRepo:   cacheRepo,
		middlewares: middlewares,
		validate:    validate,
		rabbitMq:    rabbitMq,
	}
}
