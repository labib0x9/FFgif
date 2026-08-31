package middleware

import (
	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/pkg/jwt"
)

type Middlewares struct {
	Cnf   *config.Config
	cache cache.Cache
	jwt   jwt.Jwt
}

func NewMiddlewares(cnf *config.Config, cache cache.Cache, jwt jwt.Jwt) *Middlewares {
	return &Middlewares{
		Cnf:   cnf,
		cache: cache,
		jwt:   jwt,
	}
}
