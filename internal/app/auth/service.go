//go:generate mockgen -package=mocks -destination=mocks/mock_auth_service.go github.com/labib0x9/ffgif/internal/app/auth Service

package auth

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/internal/port/db"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
)

type Service interface {
	Signup(ctx context.Context, email string, username string, fullname string, password string) (*SignupResult, error)
	Login(ctx context.Context, email string, password string) (*Result, error)
	Logout(ctx context.Context, jwt string, claims jwt.Payload) error
	ForgotPassword(ctx context.Context, email string) error
	ResendVerify(ctx context.Context, email string) error
	ResetPasswordGet(ctx context.Context, token string) (string, error)
	ResetPasswordPost(ctx context.Context, token string, pass string, confirmPass string) error
	Verify(ctx context.Context, token string) error
}

type service struct {
	authRepo     auth.AuthRepository
	verifierRepo auth.VerifierRepository
	profileRepo  user.UserRepository
	reseterRepo  auth.ReseterRepository
	quotaRepo    user.QuotaRepository
	cache        cache.Cache
	queue        queue.Queue
	jwt          jwt.Jwt
	hasher       password.Hasher
	tnx          db.TxManager
}

func NewService(
	authRepo auth.AuthRepository,
	verifierRepo auth.VerifierRepository,
	profileRepo user.UserRepository,
	reseterRepo auth.ReseterRepository,
	quotaRepo user.QuotaRepository,
	cache cache.Cache,
	queue queue.Queue,
	jwt jwt.Jwt,
	hasher password.Hasher,
	tnx db.TxManager,
) Service {
	return &service{
		authRepo:     authRepo,
		verifierRepo: verifierRepo,
		profileRepo:  profileRepo,
		reseterRepo:  reseterRepo,
		quotaRepo:    quotaRepo,
		cache:        cache,
		queue:        queue,
		jwt:          jwt,
		hasher:       hasher,
		tnx:          tnx,
	}
}

type Jwt interface {
	Create(fullname string, id string, email string, role string) (string, error)
	Verify(tokenStr string) (jwt.Payload, error)
}

type Hasher interface {
	GenerateHash(pass string) (string, error)
	CompareHashAndPassword(hashedPass string, pass string) bool
}
