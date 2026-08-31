package media

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ffgif/internal/app/media"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

type Handler struct {
	srv         media.Service
	middlewares *middleware.Middlewares
	validate    *validator.Validate
}

func NewHandler(
	srv media.Service,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		srv:         srv,
		middlewares: middlewares,
		validate:    validate,
	}
}
