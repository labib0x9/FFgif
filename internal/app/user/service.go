package user

import (
	"github.com/labib0x9/ProjectUnsafe/internal/domain/auth"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/user"
	"github.com/labib0x9/ProjectUnsafe/pkg/jwt"
	"github.com/labib0x9/ProjectUnsafe/pkg/password"
)

type Service interface {
	ChangePassword(id string, currentPass string, pass string, confirmPass string) error
	DeleteUser(id string, pass string) error
	GetProfile(id string) (user.ProfileResp, error)
	GetQuota(id string) (*user.Quota, error)
	UpdateProfile(profile user.ProfileResp, id string) (user.ProfileResp, error)
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
