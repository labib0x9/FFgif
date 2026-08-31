package share

import (
	"context"
	"time"
)

type Share struct {
	ID         string     `json:"id" db:"id"`
	GifKey     string     `json:"gif_key" db:"gif_key"`
	OwnerID    string     `json:"owner_id" db:"owner_id"`
	SharedWith string     `json:"shared_with" db:"shared_with"`
	ExpiresAt  *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

type GifResponse struct {
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
	Get(ctx context.Context, userID string) ([]GifResponse, error)
}
