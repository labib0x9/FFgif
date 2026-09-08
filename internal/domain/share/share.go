package share

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrNotAuthorized = errors.New("not authorized")
)

type Share struct {
	ID         string     `json:"id" db:"id"`
	GifKey     string     `json:"gif_key" db:"gif_key"`
	OwnerID    string     `json:"owner_id" db:"owner_id"`
	SharedWith string     `json:"shared_with" db:"shared_with"`
	ExpiresAt  *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

type ShareByToken struct {
	GifKey    string     `json:"gif_key" db:"gif_key"`
	Token     string     `json:"token"             db:"token"`
	Email     string     `json:"email"         db:"email"`
	ExpiresAt *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type GifTokenResponse struct {
	GifKey       string     `json:"gif_key" db:"gif_key"`
	Name         string     `json:"name" db:"name"`
	Url          string     `json:"url" db:"url"`
	ThumbnailUrl string     `json:"thumbnail_url" db:"thumbnail_url"`
	ExpiresAt    *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

type GifResponse struct {
	ID           string     `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Url          string     `json:"url" db:"url"`
	ThumbnailUrl string     `json:"thumbnail_url" db:"thumbnail_url"`
	GifKey       string     `json:"gif_key" db:"gif_key"`
	OwnerID      string     `json:"owner_id" db:"owner_id"`
	SharedWith   string     `json:"shared_with" db:"shared_with"`
	ExpiresAt    *time.Time `json:"expires_at" db:"expires_at"`
}

type ShareRepository interface {
	Create(ctx context.Context, gif Share) error
	CreateByToken(ctx context.Context, gif ShareByToken) error
	Get(ctx context.Context, userID string) ([]GifResponse, error)
	GetByToken(ctx context.Context, token string) (GifTokenResponse, error)
	GetOwner(ctx context.Context, user string, key string) (string, error)
	Delete(ctx context.Context, key, shareWithId string) error
}
