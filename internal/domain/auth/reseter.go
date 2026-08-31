package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Reseter struct {
	Id        int64     `json:"id" db:"id"`
	UserId    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token_hash" db:"token_hash"`
	Used      bool      `json:"used" db:"used"`
	CreatedAt time.Time `json:"created_at"    db:"created_at"`
	ExpireAt  time.Time `json:"expire_at"    db:"expire_at"`
}

type ReseterRepository interface {
	GetById(ctx context.Context, id uuid.UUID) (Reseter, error)
	Update(ctx context.Context, reseter Reseter) error
	Create(ctx context.Context, reseter Reseter) error
	GetByToken(ctx context.Context, token string) (Reseter, error)
	DeleteById(ctx context.Context, id int64) error
}
