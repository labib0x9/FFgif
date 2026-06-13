package share

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/internal/app/share"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

type Handler struct {
	srv         share.Service
	middlewares *middleware.Middlewares
	validate    *validator.Validate
}

func NewHandler(
	srv share.Service,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		srv:         srv,
		middlewares: middlewares,
		validate:    validate,
	}
}
