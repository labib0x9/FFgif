package job

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/internal/app/job"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

type Handler struct {
	srv         job.Service
	middlewares *middleware.Middlewares
	validate    *validator.Validate
}

func NewHandler(
	srv job.Service,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		srv:         srv,
		middlewares: middlewares,
		validate:    validate,
	}
}
