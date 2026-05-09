package user

import (
	"github.com/go-playground/validator/v10"
	"github.com/labib0x9/ProjectUnsafe/repo"
	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

type Handler struct {
	middlewares *middleware.Middlewares
	userRepo    repo.UserRepository
	quotaRepo   repo.QuotaRepository
	authRepo    repo.AuthRepository
	validate    *validator.Validate
}

func NewHandler(
	userRepo repo.UserRepository,
	quotaRepo repo.QuotaRepository,
	authRepo repo.AuthRepository,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		userRepo:    userRepo,
		quotaRepo:   quotaRepo,
		authRepo:    authRepo,
		middlewares: middlewares,
		validate:    validate,
	}
}
