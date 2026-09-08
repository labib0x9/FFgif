//go:generate mockgen -package=mocks -destination=mocks/mock_verifier_repo.go github.com/labib0x9/ffgif/internal/domain/auth VerifierRepository

package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrVerifierTokenCreateFailed = errors.New("verifier token create failed")
	ErrSetProfileFailed          = errors.New("set profile failed")
	ErrQuotaCreateFailed         = errors.New("quota create failed")
	ErrTokenFetchFailed          = errors.New("token fetch failed")
	ErrCreateResetTokenFailed    = errors.New("reset token create failed")
	ErrUserAlreadyVerified       = errors.New("user already verified")
	ErrReseterTokenFatchFailed   = errors.New("reset token fetch failed")
	ErrUserFetchError            = errors.New("user fetch failed")
	ErrInvalidToken              = errors.New("invalid token")
	ErrSetUserVerifiedFailed     = errors.New("set user verified failed")
)

type Verifier struct {
	Id        int64     `json:"id" db:"id"`
	UserId    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token_hash" db:"token_hash"`
	CreatedAt time.Time `json:"created_at"    db:"created_at"`
	ExpireAt  time.Time `json:"expire_at"    db:"expire_at"`
}

type VerifierRepository interface {
	Create(ctx context.Context, verifier Verifier) error
	GetByHash(ctx context.Context, tokenHash string) (Verifier, error)
	Delete(ctx context.Context, id int64) error
	GetById(ctx context.Context, userId uuid.UUID) (Verifier, error)
}
