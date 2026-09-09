package media

import (
	"context"
	"errors"
	"time"
)

var (
	ErrGifNotFound          = errors.New("gif not found")
	ErrGifFetchFailed       = errors.New("gif fetch failed")
	ErrGifOwnerMismatch     = errors.New("Gif Owner mismatch")
	ErrThumbnailNotFound    = errors.New("thumbnail not found")
	ErrETagValidationFailed = errors.New("etag validation failed")
)

type Gif struct {
	Key          string    `json:"key" db:"key"`
	Name         string    `json:"name" db:"name"`
	UserId       string    `json:"user_id" db:"user_id"`
	Status       string    `json:"status" db:"status"`
	Persist      bool      `json:"persist" db:"persist"`
	Download     int       `json:"download" db:"download"`
	Url          string    `json:"url" db:"url"`
	ThumbnailUrl string    `json:"thumbnail_url" db:"thumbnail_url"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}

type GifResponse struct {
	Key          string    `json:"key" db:"key"`
	Name         string    `json:"name" db:"name"`
	Status       string    `json:"status" db:"status"`
	Persist      bool      `json:"persist" db:"persist"`
	Url          string    `json:"url" db:"url"`
	ThumbnailUrl string    `json:"thumbnail_url" db:"thumbnail_url"`
	Download     int       `json:"download" db:"download"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}

type GifUpdateRequest struct {
	Name    *string `json:"name,omitempty" db:"name"`
	Status  *string `json:"status,omitempty" db:"status"`
	Persist *bool   `json:"persist,omitempty" db:"persist"`
}

//go:generate mockgen -source=gif.go -destination=mocks/mock_gif_repository.go -package=mocks
type GifRepository interface {
	Create(ctx context.Context, gif Gif) error
	Get(ctx context.Context, user_id string, status string) ([]GifResponse, error)
	GetByKey(ctx context.Context, key string, forUpdate bool) (GifResponse, error)
	GetRecents(ctx context.Context, user_id string) ([]GifResponse, error)
	Delete(ctx context.Context, key string) error
	Update(ctx context.Context, key string, req GifUpdateRequest) (GifResponse, error)
	SaveRecent(ctx context.Context, key string) error
	GetOwner(ctx context.Context, key string) (string, error)
}
