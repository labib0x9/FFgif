package user

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/internal/app/user"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

type Handler struct {
	srv         user.Service
	middlewares *middleware.Middlewares
	validate    *validator.Validate
}

func NewHandler(
	srv user.Service,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		srv:         srv,
		middlewares: middlewares,
		validate:    validate,
	}
}
