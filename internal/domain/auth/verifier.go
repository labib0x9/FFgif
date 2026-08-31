package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrVerifierTokenCreateFailed = errors.New("verifier create failed")
	ErrSetProfileFailed          = errors.New("Set profile failed")
	ErrQuotaCreateFailed         = errors.New("quota create failed")
	ErrMessageQueueFailed        = errors.New("message queue failed")
	ErrTokenFetchFailed          = errors.New("token fetch failed")
	ErrCreateResetTokenFailed    = errors.New("")
	ErrUserAlreadyVerified       = errors.New("")
	ErrReseterTokenFatchFailed   = errors.New("")
	ErrUserFetchError            = errors.New("")
	ErrUserTableUpdateFailed     = errors.New("")
	ErrInvalidToken              = errors.New("")
	ErrSetUserVerifiedFailed     = errors.New("")
	ErrPasswordMismatched        = errors.New("")
	// ErrHashGenFailed             = errors.New("")
	ErrTableUpdateFailed = errors.New("")
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
