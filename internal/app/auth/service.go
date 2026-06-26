package auth

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/jwt"
	"github.com/labib0x9/ffgif/pkg/password"
)

type Service interface {
	Signup(ctx context.Context, email string, username string, fullname string, password string) (*SignupResult, error)
	Login(email string, password string) (*Result, error)
	Logout(ctx context.Context, jwt string, claims jwt.Payload) error
	ForgotPassword(ctx context.Context, email string) error
	ResendVerify(ctx context.Context, email string) error
	ResetPasswordGet(token string) (string, error)
	ResetPasswordPost(ctx context.Context, token string, pass string, confirmPass string) error
	Verify(token string) error
}

type service struct {
	authRepo     auth.AuthRepository
	verifierRepo auth.VerifierRepo
	profileRepo  user.UserRepository
	reseterRepo  auth.ReseterRepo
	quotaRepo    user.QuotaRepository
	cache        cache.Cache
	queue        queue.EmailQueue
	jwt          jwt.Jwt
	hasher       password.Hasher
}

func NewService(
	authRepo auth.AuthRepository,
	verifierRepo auth.VerifierRepo,
	profileRepo user.UserRepository,
	reseterRepo auth.ReseterRepo,
	quotaRepo user.QuotaRepository,
	cache cache.Cache,
	queue queue.EmailQueue,
	jwt jwt.Jwt,
	hasher password.Hasher,
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
