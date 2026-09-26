package friend

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ffgif/internal/app/friend"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

type Handler struct {
	srv         friend.Service
	middlewares *middleware.Middlewares
	validate    *validator.Validate
}

func NewHandler(srv friend.Service, middlewares *middleware.Middlewares, validate *validator.Validate) *Handler {
	return &Handler{srv: srv, middlewares: middlewares, validate: validate}
}
