package user

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/db"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
)

type Service interface {
	ChangePassword(ctx context.Context, userId string, currentPass string, pass string, confirmPass string) error
	DeleteUser(ctx context.Context, userId string, pass string) error
	GetProfile(ctx context.Context, userId string) (*user.ProfileResponse, error)
	GetQuota(ctx context.Context, userId string) (*user.Quota, error)
	UpdateProfile(ctx context.Context, profile user.ProfileUpdateRequest, userId, lastUpdatedAt string) (*user.ProfileResponse, error)
}

type service struct {
	userRepo  user.UserRepository
	quotaRepo user.QuotaRepository
	authRepo  auth.AuthRepository
	tnx       db.TxManager
	jwt       jwt.Jwt
	hasher    password.Hasher
}

func NewService(
	userRepo user.UserRepository,
	quotaRepo user.QuotaRepository,
	authRepo auth.AuthRepository,
	tnx db.TxManager,
	jwt jwt.Jwt,
	hasher password.Hasher,
) Service {
	return &service{
		userRepo:  userRepo,
		quotaRepo: quotaRepo,
		authRepo:  authRepo,
		tnx:       tnx,
		jwt:       jwt,
		hasher:    hasher,
	}
}
