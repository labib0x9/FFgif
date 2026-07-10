package user

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
)

type Service interface {
	ChangePassword(ctx context.Context, id string, currentPass string, pass string, confirmPass string) error
	DeleteUser(ctx context.Context, id string, pass string) error
	GetProfile(ctx context.Context, id string) (user.ProfileResp, error)
	GetQuota(ctx context.Context, id string) (*user.Quota, error)
	UpdateProfile(ctx context.Context, profile user.ProfileResp, id string) (user.ProfileResp, error)
}

type service struct {
	userRepo  user.UserRepository
	quotaRepo user.QuotaRepository
	authRepo  auth.AuthRepository
	jwt       jwt.Jwt
	hasher    password.Hasher
}

func NewService(
	userRepo user.UserRepository,
	quotaRepo user.QuotaRepository,
	authRepo auth.AuthRepository,
	jwt jwt.Jwt,
	hasher password.Hasher,
) Service {
	return &service{
		userRepo:  userRepo,
		quotaRepo: quotaRepo,
		authRepo:  authRepo,
		jwt:       jwt,
		hasher:    hasher,
	}
}
