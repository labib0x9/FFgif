package gif

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
	gifRepo     repo.GifRepository
	validate    *validator.Validate
}

func NewHandler(
	userRepo repo.UserRepository,
	quotaRepo repo.QuotaRepository,
	authRepo repo.AuthRepository,
	gifRepo repo.GifRepository,
	middlewares *middleware.Middlewares,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		userRepo:    userRepo,
		quotaRepo:   quotaRepo,
		authRepo:    authRepo,
		gifRepo:     gifRepo,
		middlewares: middlewares,
		validate:    validate,
	}
}
