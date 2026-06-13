package middleware

import (
	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/cache"
	"github.com/labib0x9/ProjectUnsafe/pkg/jwt"
)

type Middlewares struct {
	Cnf   *config.Config
	cache cache.CacheRepo
	jwt   jwt.Jwt
}

func NewMiddlewares(cnf *config.Config, cache cache.CacheRepo, jwt jwt.Jwt) *Middlewares {
	return &Middlewares{
		Cnf:   cnf,
		cache: cache,
		jwt:   jwt,
	}
}
