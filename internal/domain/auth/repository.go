package auth

import (
	"context"

	"github.com/google/uuid"
)

type AuthRepository interface {
	GetByEmail(ctx context.Context, email string) (User, error)
	GetById(ctx context.Context, id uuid.UUID) (User, error)
	Create(ctx context.Context, user User) (User, error)
	DeleteById(ctx context.Context, id uuid.UUID) error
	DeleteByEmail(ctx context.Context, email string) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error
	SetVerified(ctx context.Context, userId uuid.UUID) error
	Upgrade(ctx context.Context, id string, user User) (User, error)
	// CreateDemo(user AnonUser) (AnonUser, error)
}

type VerifierRepo interface {
	Create(ctx context.Context, verifier Verifier) error
	GetByHash(ctx context.Context, tokenHash string) (Verifier, error)
	Delete(ctx context.Context, id int64) error
	GetById(ctx context.Context, userId uuid.UUID) (Verifier, error)
}

type ReseterRepo interface {
	GetById(ctx context.Context, id uuid.UUID) (Reseter, error)
	Update(ctx context.Context, reseter Reseter) error
	Create(ctx context.Context, reseter Reseter) error
	GetByToken(ctx context.Context, token string) (Reseter, error)
	DeleteById(ctx context.Context, id int64) error
}
