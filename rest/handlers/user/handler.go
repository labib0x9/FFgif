package user

import (
	"github.com/labib0x9/ProjectUnsafe/repo"
	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

type Handler struct {
	middlewares *middleware.Middlewares
	userRepo    repo.UserRepository
	quotaRepo   repo.QuotaRepository
}

func NewHandler(userRepo repo.UserRepository, quotaRepo repo.QuotaRepository, middlewares *middleware.Middlewares) *Handler {
	return &Handler{
		userRepo:    userRepo,
		quotaRepo:   quotaRepo,
		middlewares: middlewares,
	}
}
